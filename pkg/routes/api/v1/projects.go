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
	"fmt"
	"net/http"

	"github.com/FerrPOINT/task-bord/pkg/models"
	"github.com/FerrPOINT/task-bord/pkg/web/handler"

	"github.com/danielgtaylor/huma/v2"
)

// projectListBody is the list-response envelope. models.Project.ReadAll
// returns []*models.Project, so that's the element type.
type projectListBody struct {
	Body Paginated[*models.Project]
}

// RegisterProjectRoutes wires Project CRUD onto the Huma API.
func RegisterProjectRoutes(api huma.API) {
	tags := []string{"projects"}

	Register(api, huma.Operation{
		OperationID: "projects-list",
		Summary:     "List projects",
		Description: "Returns the projects the authenticated user has access to (owned plus shared, with child projects of accessible parents), paginated. Archived projects are excluded unless is_archived=true. Pass expand=permissions to include each project's max_permission for the caller.",
		Method:      http.MethodGet,
		Path:        "/projects",
		Tags:        tags,
	}, projectsList)

	Register(api, huma.Operation{
		OperationID: "projects-read",
		Summary:     "Get a project",
		Description: "Returns a single project the caller can read, including its views, the caller's favorite/subscription state and the caller's max_permission. Resolves the Favorites pseudo-project and saved-filter-backed projects. Served fresh on every call (no conditional/ETag) because the response carries user-scoped state that changes without bumping the project's updated timestamp.",
		Method:      http.MethodGet,
		Path:        "/projects/{id}",
		Tags:        tags,
	}, projectsRead)

	Register(api, huma.Operation{
		OperationID: "projects-create",
		Summary:     "Create a project",
		Description: "Creates a project; the authenticated user becomes its owner. When parent_project_id is set, the caller needs write access to that parent. Default views and a backlog bucket are created automatically.",
		Method:      http.MethodPost,
		Path:        "/projects",
		Tags:        tags,
	}, projectsCreate)

	Register(api, huma.Operation{
		OperationID: "projects-update",
		Summary:     "Update a project",
		Description: "Replaces a project's fields. Requires write access (admin to reparent or delete). Use PATCH for a partial update.",
		Method:      http.MethodPut,
		Path:        "/projects/{id}",
		Tags:        tags,
	}, projectsUpdate)

	Register(api, huma.Operation{
		OperationID: "projects-delete",
		Summary:     "Delete a project",
		Description: "Deletes a project together with its tasks, views, buckets and child projects. Only project admins may delete it.",
		Method:      http.MethodDelete,
		Path:        "/projects/{id}",
		Tags:        tags,
	}, projectsDelete)
}

func init() { AddRouteRegistrar(RegisterProjectRoutes) }

func projectsList(ctx context.Context, in *struct {
	ListParams
	IsArchived bool `query:"is_archived" doc:"If true, also returns archived projects."`
}) (*projectListBody, error) {
	a, err := authFromCtx(ctx)
	if err != nil {
		return nil, err
	}
	p := &models.Project{
		IsArchived: in.IsArchived,
	}
	result, _, total, err := handler.DoReadAll(ctx, p, a, in.Q, in.Page, in.PerPage)
	if err != nil {
		return nil, translateDomainError(err)
	}
	items, ok := result.([]*models.Project)
	if !ok {
		return nil, fmt.Errorf("projects.ReadAll returned unexpected type %T (expected []*models.Project)", result)
	}
	return &projectListBody{Body: NewPaginated(items, total, in.Page, in.PerPage)}, nil
}

// projectReadBody is the read shape.
type projectReadBody struct {
	models.Project
}

func projectsRead(ctx context.Context, in *struct {
	ID int64 `path:"id"`
}) (*singleBody[projectReadBody], error) {
	a, err := authFromCtx(ctx)
	if err != nil {
		return nil, err
	}
	project := &models.Project{ID: in.ID}
	_, err = handler.DoReadOne(ctx, project, a)
	if err != nil {
		return nil, translateDomainError(err)
	}
	return &singleBody[projectReadBody]{Body: &projectReadBody{Project: *project}}, nil
}

func projectsCreate(ctx context.Context, in *struct {
	Body models.Project
}) (*singleBody[models.Project], error) {
	a, err := authFromCtx(ctx)
	if err != nil {
		return nil, err
	}
	if err := handler.DoCreate(ctx, &in.Body, a); err != nil {
		return nil, translateDomainError(err)
	}
	return &singleBody[models.Project]{Body: &in.Body}, nil
}

// Body matches the read shape so AutoPatch's GET→PUT echo validates.
func projectsUpdate(ctx context.Context, in *struct {
	ID   int64 `path:"id"`
	Body projectReadBody
}) (*singleBody[models.Project], error) {
	a, err := authFromCtx(ctx)
	if err != nil {
		return nil, err
	}
	project := &in.Body.Project
	project.ID = in.ID // URL wins over body
	if err := handler.DoUpdate(ctx, project, a); err != nil {
		return nil, translateDomainError(err)
	}
	return &singleBody[models.Project]{Body: project}, nil
}

func projectsDelete(ctx context.Context, in *struct {
	ID int64 `path:"id"`
}) (*emptyBody, error) {
	a, err := authFromCtx(ctx)
	if err != nil {
		return nil, err
	}
	if err := handler.DoDelete(ctx, &models.Project{ID: in.ID}, a); err != nil {
		return nil, translateDomainError(err)
	}
	return &emptyBody{}, nil
}
