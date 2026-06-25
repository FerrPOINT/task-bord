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

package models

import (
	"github.com/FerrPOINT/task-bord/pkg/user"
	"github.com/FerrPOINT/task-bord/pkg/web"

	"xorm.io/xorm"
)

// isInstanceAdmin gates cross-user access on is_admin.
// is_admin is re-read from the DB because the auth's flag is claim-derived and
// stale until the JWT expires.
func isInstanceAdmin(s *xorm.Session, a web.Auth) bool {
	u, ok := a.(*user.User)
	if !ok {
		return false
	}
	fresh, err := user.GetUserByID(s, u.ID)
	if err != nil {
		return false
	}
	return fresh.IsAdmin
}
