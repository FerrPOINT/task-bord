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

// Package shared holds helpers used by both the v1 and v2 route packages. It
// sits above the auth/user modules in the import graph, so it can combine them
// without creating a cycle.
package shared

import (
	"github.com/FerrPOINT/task-bord/pkg/user"
)

// GetAuthProviderName resolves the human-readable name of the source a user
// authenticated with. MVP only supports local accounts.
func GetAuthProviderName(u *user.User) (string, error) {
	if u.Issuer == user.IssuerLocal || u.Issuer == "" {
		return "local", nil
	}
	return "", nil
}
