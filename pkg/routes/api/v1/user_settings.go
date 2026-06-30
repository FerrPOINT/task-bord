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

	"github.com/FerrPOINT/task-bord/pkg/db"
	"github.com/FerrPOINT/task-bord/pkg/models"
	"github.com/FerrPOINT/task-bord/pkg/user"

	"github.com/danielgtaylor/huma/v2"
)

type userInfoBody struct {
	user.User
	IsLocalUser bool `json:"is_local_user" readOnly:"true"`
}

type userActionMessageBody struct {
	Message string `json:"message" readOnly:"true" doc:"A confirmation message."`
}

// RegisterUserSettingsRoutes wires the current-user account endpoints.
func RegisterUserSettingsRoutes(api huma.API) {
	tags := []string{"user"}

	Register(api, huma.Operation{
		OperationID: "user-show",
		Summary:     "Get the current user",
		Description: "Returns the authenticated user.",
		Method:      http.MethodGet,
		Path:        "/user",
		Tags:        tags,
	}, userShow)

	Register(api, huma.Operation{
		OperationID:   "user-change-password",
		Summary:       "Change the current user's password",
		Description:   "Changes the authenticated user's password after verifying the old one.",
		Method:        http.MethodPost,
		Path:          "/user/password",
		DefaultStatus: http.StatusOK,
		Tags:          tags,
	}, userChangePassword)

	Register(api, huma.Operation{
		OperationID: "user-update-settings",
		Summary:     "Update the current user's general settings",
		Description: "Replaces the authenticated user's general settings (name, language, timezone).",
		Method:      http.MethodPut,
		Path:        "/user/settings/general",
		Tags:        tags,
	}, userUpdateSettings)
}

func init() { AddRouteRegistrar(RegisterUserSettingsRoutes) }

func userShow(ctx context.Context, _ *struct{}) (*singleBody[userInfoBody], error) {
	a, err := authFromCtx(ctx)
	if err != nil {
		return nil, err
	}

	s := db.NewSession()
	defer s.Close()

	u, err := models.GetUserByAuth(s, a)
	if err != nil {
		return nil, translateDomainError(err)
	}

	info := &userInfoBody{
		User:        *u,
		IsLocalUser: u.Issuer == user.IssuerLocal,
	}

	return &singleBody[userInfoBody]{Body: info}, nil
}

func userChangePassword(ctx context.Context, in *struct {
	Body struct {
		OldPassword string `json:"old_password" doc:"The current password, for confirmation."`
		NewPassword string `json:"new_password" valid:"bcrypt_password" minLength:"1" maxLength:"72" doc:"The new password. Max 72 bytes (a bcrypt limit), which may be fewer than 72 characters."`
	}
}) (*singleBody[userActionMessageBody], error) {
	a, err := authFromCtx(ctx)
	if err != nil {
		return nil, err
	}
	doer, err := user.GetFromAuth(a)
	if err != nil {
		return nil, translateDomainError(err)
	}

	s := db.NewSession()
	defer s.Close()

	if err := models.ChangeUserPassword(ctx, s, doer, in.Body.OldPassword, in.Body.NewPassword); err != nil {
		_ = s.Rollback()
		return nil, translateDomainError(err)
	}

	if err := s.Commit(); err != nil {
		return nil, translateDomainError(err)
	}

	return &singleBody[userActionMessageBody]{Body: &userActionMessageBody{Message: "The password was updated successfully."}}, nil
}

func userUpdateSettings(ctx context.Context, in *struct {
	Body models.UserGeneralSettings
}) (*singleBody[userActionMessageBody], error) {
	a, err := authFromCtx(ctx)
	if err != nil {
		return nil, err
	}
	doer, err := user.GetFromAuth(a)
	if err != nil {
		return nil, translateDomainError(err)
	}

	s := db.NewSession()
	defer s.Close()

	u, err := user.GetUserWithEmail(s, &user.User{ID: doer.ID})
	if err != nil {
		_ = s.Rollback()
		return nil, translateDomainError(err)
	}

	if err := models.UpdateUserGeneralSettings(s, u, &in.Body); err != nil {
		_ = s.Rollback()
		return nil, translateDomainError(err)
	}

	if err := s.Commit(); err != nil {
		return nil, translateDomainError(err)
	}

	return &singleBody[userActionMessageBody]{Body: &userActionMessageBody{Message: "The settings were updated successfully."}}, nil
}

func userTimezones(ctx context.Context, _ *struct{}) (*timezonesBody, error) {
	if _, err := authFromCtx(ctx); err != nil {
		return nil, err
	}
	return &timezonesBody{Body: []string{"UTC"}}, nil
}

type timezonesBody struct {
	Body []string
}
