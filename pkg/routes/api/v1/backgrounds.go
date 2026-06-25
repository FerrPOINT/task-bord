// Task Board is a self-hosted Kanban application.
// Copyright 2026-present Task Board contributors. All rights reserved.
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program.  If not, see <https://www.gnu.org/licenses/>.

package apiv1

import (
	"context"
	"net/http"

	"github.com/FerrPOINT/task-bord/pkg/config"
	"github.com/FerrPOINT/task-bord/pkg/db"
	"github.com/FerrPOINT/task-bord/pkg/models"
	backgroundHandler "github.com/FerrPOINT/task-bord/pkg/modules/background/handler"
	"github.com/FerrPOINT/task-bord/pkg/modules/humaecho5"
	webfiles "github.com/FerrPOINT/task-bord/pkg/web/files"

	"github.com/danielgtaylor/huma/v2"
)

// RegisterBackgroundRoutes wires the project-background actions onto the Huma
// API. BackgroundsEnabled is static config, so the registrar early-returns
// instead of gating per request.
func RegisterBackgroundRoutes(api huma.API) {
	if !config.BackgroundsEnabled.GetBool() {
		return
	}

	tags := []string{"project"}

	Register(api, huma.Operation{
		OperationID: "projects-background-delete",
		Summary:     "Remove a project background",
		Description: "Removes a project's background, whichever provider set it. Succeeds even when the project has no background. Requires write access to the project. Returns the updated project.",
		Method:      http.MethodDelete,
		Path:        "/projects/{project}/background",
		// Return the updated project with 200, not the wrapper's DELETE default 204.
		DefaultStatus: http.StatusOK,
		Tags:          tags,
	}, backgroundRemove)

	Register(api, huma.Operation{
		OperationID: "projects-background-get",
		Summary:     "Get a project background",
		Description: "Streams a project's background image, whichever provider set it. Requires read access to the project. Always served as image/jpeg with a revalidation Last-Modified header, so a conditional If-Modified-Since request gets a 304. Returns 404 when the project has no background.",
		Method:      http.MethodGet,
		Path:        "/projects/{project}/background",
		Tags:        tags,
		// Spell out the binary response; the default would be modeled as JSON.
		Responses: map[string]*huma.Response{
			"200": {
				Description: "The project background as a jpeg image.",
				Content: map[string]*huma.MediaType{
					"image/jpeg": {
						Schema: &huma.Schema{Type: huma.TypeString, Format: "binary"},
					},
				},
			},
		},
	}, backgroundGet)

	if config.BackgroundsUploadEnabled.GetBool() {
		Register(api, huma.Operation{
			OperationID: "projects-background-upload",
			Summary:     "Upload a project background",
			Description: "Uploads an image via multipart/form-data under the \"background\" field and sets it as the project's background. Requires write access to the project. The image is resized server-side and stored as JPEG; it replaces any previous background (idempotent replace, hence PUT). Returns the updated project.",
			Method:      http.MethodPut,
			Path:        "/projects/{project}/backgrounds/upload",
			// Return the updated project with 200, the natural code for an idempotent PUT.
			DefaultStatus: http.StatusOK,
			Tags:          tags,
			// +2 MB mirrors Echo's global BodyLimit overhead so a max-sized file isn't rejected by multipart boundary/header bytes.
			// #nosec G115 - configured value won't exceed int64 max in practice.
			MaxBodyBytes: (int64(config.GetMaxFileSizeInMBytes()) + 2) * 1024 * 1024,
		}, backgroundUpload)
	}
}

func init() { AddRouteRegistrar(RegisterBackgroundRoutes) }

type backgroundUploadInput struct {
	ProjectID int64 `path:"project" doc:"The id of the project to set the background on."`
	// Allow-list mirrors the formats background uploads can actually be decoded as
	// (handler.ValidateAndSaveBackgroundUpload's allowedImageMimes); octet-stream covers
	// programmatic clients. Huma's MimeTypeValidator rejects the part pre-handler, so the
	// byte-level image check in the shared function is the real gate.
	RawBody huma.MultipartFormFiles[struct {
		Background huma.FormFile `form:"background" contentType:"image/jpeg,image/png,image/gif,image/bmp,image/tiff,image/webp,application/octet-stream" required:"true" doc:"The background image to upload. Must be a decodable raster image (JPEG, PNG, GIF, BMP, TIFF or WebP); it is resized server-side and re-encoded as JPEG."`
	}]
}

// backgroundUpload owns auth, the session and the permission check because there is
// no handler.Do* for multipart uploads (see the api-v2-routes skill's "Non-CRUDable
// / custom routes" section). It shares its body with v1 via
// handler.ValidateAndSaveBackgroundUpload.
func backgroundUpload(ctx context.Context, in *backgroundUploadInput) (*singleBody[models.Project], error) {
	a, err := authFromCtx(ctx)
	if err != nil {
		return nil, err
	}

	s := db.NewSession()
	defer s.Close()

	project := &models.Project{ID: in.ProjectID}
	can, err := project.CanUpdate(s, a)
	if err != nil {
		_ = s.Rollback()
		return nil, translateDomainError(err)
	}
	if !can {
		_ = s.Rollback()
		return nil, huma.Error403Forbidden("forbidden")
	}
	project, err = models.GetProjectSimpleByID(s, in.ProjectID)
	if err != nil {
		_ = s.Rollback()
		return nil, translateDomainError(err)
	}

	file := in.RawBody.Data().Background
	defer func() { _ = file.Close() }()

	if err := backgroundHandler.ValidateAndSaveBackgroundUpload(s, a, project, file, file.Filename, uint64(file.Size)); err != nil {
		_ = s.Rollback()
		if backgroundHandler.IsErrFileIsNoImage(err) || backgroundHandler.IsErrFileUnsupportedImageFormat(err) {
			return nil, huma.Error400BadRequest(err.Error())
		}
		return nil, translateDomainError(err)
	}

	if err := s.Commit(); err != nil {
		return nil, translateDomainError(err)
	}

	return &singleBody[models.Project]{Body: project}, nil
}

// backgroundGet owns auth, the session and the permission check because there is no
// handler.Do* for a file body. CanRead hydrates the project (including its
// BackgroundFileID), which the shared loader then needs.
func backgroundGet(ctx context.Context, in *struct {
	ProjectID int64 `path:"project" doc:"The id of the project whose background to fetch."`
}) (*huma.StreamResponse, error) {
	a, err := authFromCtx(ctx)
	if err != nil {
		return nil, err
	}

	s := db.NewSession()
	defer s.Close()

	project := &models.Project{ID: in.ProjectID}
	can, _, err := project.CanRead(s, a)
	if err != nil {
		_ = s.Rollback()
		return nil, translateDomainError(err)
	}
	if !can {
		_ = s.Rollback()
		return nil, huma.Error403Forbidden("forbidden")
	}

	bgFile, stat, err := backgroundHandler.LoadProjectBackgroundForDownload(s, project)
	if err != nil {
		_ = s.Rollback()
		return nil, translateDomainError(err)
	}

	// The file reader comes from object storage, not the DB session, so it stays
	// valid after the commit; the StreamResponse callback runs after this returns.
	if err := s.Commit(); err != nil {
		_ = s.Rollback()
		// The stream callback (which closes the reader) won't run on this error path.
		_ = bgFile.File.Close()
		return nil, translateDomainError(err)
	}

	return &huma.StreamResponse{Body: func(hctx huma.Context) {
		defer func() { _ = bgFile.File.Close() }()
		c := humaecho5.Unwrap(hctx)
		webfiles.WriteProjectBackground((*c).Response(), (*c).Request(), bgFile, stat)
	}}, nil
}

func backgroundRemove(ctx context.Context, in *struct {
	ProjectID int64 `path:"project"`
}) (*singleBody[models.Project], error) {
	a, err := authFromCtx(ctx)
	if err != nil {
		return nil, err
	}

	s := db.NewSession()
	defer s.Close()

	project := &models.Project{ID: in.ProjectID}
	can, err := project.CanUpdate(s, a)
	if err != nil {
		_ = s.Rollback()
		return nil, translateDomainError(err)
	}
	if !can {
		_ = s.Rollback()
		return nil, huma.Error403Forbidden("forbidden")
	}

	if err := project.DeleteBackgroundFileIfExists(s); err != nil {
		_ = s.Rollback()
		return nil, translateDomainError(err)
	}
	if err := models.ClearProjectBackground(s, project.ID); err != nil {
		_ = s.Rollback()
		return nil, translateDomainError(err)
	}
	if err := s.Commit(); err != nil {
		return nil, translateDomainError(err)
	}

	return &singleBody[models.Project]{Body: project}, nil
}
