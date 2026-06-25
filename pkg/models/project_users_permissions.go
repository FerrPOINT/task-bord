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
	"github.com/FerrPOINT/task-bord/pkg/web"
	"xorm.io/xorm"
)

// CanCreate checks if the user can create a new user <-> project relation
func (lu *ProjectUser) CanCreate(s *xorm.Session, a web.Auth) (bool, error) {
	return lu.canDoProjectUser(s, a)
}

// CanDelete checks if the user can delete a user <-> project relation
func (lu *ProjectUser) CanDelete(s *xorm.Session, a web.Auth) (bool, error) {
	return lu.canDoProjectUser(s, a)
}

// CanUpdate checks if the user can update a user <-> project relation
func (lu *ProjectUser) CanUpdate(s *xorm.Session, a web.Auth) (bool, error) {
	return lu.canDoProjectUser(s, a)
}

func (lu *ProjectUser) canDoProjectUser(s *xorm.Session, a web.Auth) (bool, error) {
	if isInstanceAdmin(s, a) {
		return true, nil
	}

	// Get the project and check if the user has write access on it
	l := Project{ID: lu.ProjectID}
	return l.IsAdmin(s, a)
}
