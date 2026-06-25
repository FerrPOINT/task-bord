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

package shared

import (
	"github.com/FerrPOINT/task-bord/pkg/user"
)

// AdminUser re-exposes fields hidden by the default user.User JSON view.
type AdminUser struct {
	*user.User
	IsAdmin      bool        `json:"is_admin" readOnly:"true" doc:"Whether the user is an instance admin."`
	Status       user.Status `json:"status" readOnly:"true" doc:"Account status (0=active, 1=email-confirmation required, 2=disabled, 3=locked)."`
	Issuer       string      `json:"issuer" readOnly:"true" doc:"Authentication issuer; empty or 'local' for local accounts."`
	Subject      string      `json:"subject,omitempty" readOnly:"true" doc:"External subject identifier, for non-local accounts."`
	AuthProvider string      `json:"auth_provider,omitempty" readOnly:"true" doc:"Resolved auth provider name, empty for local accounts."`
}

// NewAdminUser builds the admin-facing user view.
func NewAdminUser(u *user.User, providers []*user.User) *AdminUser {
	_ = providers // external auth providers removed for the MVP
	return &AdminUser{
		User:         u,
		IsAdmin:      u.IsAdmin,
		Status:       u.Status,
		Issuer:       u.Issuer,
		Subject:      u.Subject,
		AuthProvider: "",
	}
}
