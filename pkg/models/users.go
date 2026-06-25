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
	"fmt"

	"github.com/FerrPOINT/task-bord/pkg/user"
	"github.com/FerrPOINT/task-bord/pkg/web"
	"xorm.io/xorm"
)

// doerFromAuth resolves the authenticated principal into a full user for event payloads. The JWT
// only carries id + username, so without a re-fetch notifications and emails render the
// auto-generated username instead of the display name (#2720). Status errors (disabled/locked) are
// swallowed because their user is still populated and some flows act on behalf of such accounts
// (e.g. user deletion deletes that user's tasks); the partial principal is used as a last resort.
func doerFromAuth(s *xorm.Session, a web.Auth) *user.User {
	if a == nil {
		return nil
	}

	doer, err := GetUserByAuth(s, a)
	if err != nil && !user.IsErrUserStatusError(err) {
		doer = nil
	}
	if doer != nil && doer.ID != 0 {
		return doer
	}

	if u, is := a.(*user.User); is {
		return u
	}
	return &user.User{ID: a.GetID()}
}

// GetUserByAuth returns the full user for an auth principal.
func GetUserByAuth(s *xorm.Session, a web.Auth) (uu *user.User, err error) {
	if u, is := a.(*user.User); is {
		uu, err = user.GetUserByID(s, u.ID)
		return
	}
	return nil, fmt.Errorf("auth principal is not a user")
}

// getUsersByIDsIgnoringLinkShares returns all users from a slice of ids,
// ignoring negative ids which were historically used for link shares.
func getUsersByIDsIgnoringLinkShares(s *xorm.Session, ids []int64) (users map[int64]*user.User, err error) {
	users = make(map[int64]*user.User)
	var userIDs []int64
	for _, id := range ids {
		if id < 0 {
			continue
		}
		userIDs = append(userIDs, id)
	}

	if len(userIDs) > 0 {
		users, err = user.GetUsersByIDs(s, userIDs)
	}
	return
}
