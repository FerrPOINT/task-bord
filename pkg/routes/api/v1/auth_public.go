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
	"github.com/FerrPOINT/task-bord/pkg/routes/api/shared"
	"github.com/FerrPOINT/task-bord/pkg/user"

	"github.com/danielgtaylor/huma/v2"
)

// publicSecurity is the empty security requirement that opts an operation out of
// the globally-applied JWT/API-token auth. The matching Echo path must also be
// listed in unauthenticatedAPIPaths so the token middleware lets it through.
var publicSecurity = []map[string][]string{}

// registerUserBody is the response wrapper for the registration endpoint.
type registerUserBody struct {
	Body *user.User
}

func init() { AddRouteRegistrar(RegisterPublicAuthRoutes) }

// RegisterPublicAuthRoutes wires the unauthenticated local-account flows.
func RegisterPublicAuthRoutes(api huma.API) {
	if config.AuthLocalEnabled.GetBool() {
		registerLocalAuthRoutes(api)
	}
}

func registerLocalAuthRoutes(api huma.API) {
	authTags := []string{"auth"}

	if config.ServiceEnableRegistration.GetBool() {
		Register(api, huma.Operation{
			OperationID: "auth-register",
			Summary:     "Register",
			Description: "Creates a new local user account.",
			Method:      http.MethodPost,
			Path:        "/register",
			Tags:        authTags,
			Security:    publicSecurity,
		}, authRegister)
	}
}

func authRegister(ctx context.Context, in *struct{ Body shared.UserRegister }) (*registerUserBody, error) {
	newUser, err := shared.RegisterUser(ctx, &in.Body)
	if err != nil {
		return nil, translateDomainError(err)
	}
	return &registerUserBody{Body: newUser}, nil
}
