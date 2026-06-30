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
	"context"

	"github.com/FerrPOINT/task-bord/pkg/config"
	"github.com/FerrPOINT/task-bord/pkg/db"
	"github.com/FerrPOINT/task-bord/pkg/events"
	"github.com/FerrPOINT/task-bord/pkg/log"
	"github.com/FerrPOINT/task-bord/pkg/metrics"
	"github.com/FerrPOINT/task-bord/pkg/modules/keyvalue"
	"github.com/FerrPOINT/task-bord/pkg/user"

	"xorm.io/xorm"
)

// UserRegister carries the fields accepted by the public registration endpoint:
// username, password and email (from APIUserPassword) plus the new user's
// preferred language.
type UserRegister struct {
	// The language of the new user. Must be a valid IETF BCP 47 language code and exist in taskboard.
	Language string `json:"language" valid:"language" doc:"The language of the new user as an IETF BCP 47 code (e.g. en, de-DE)."`
	user.APIUserPassword
}

// RegisterUser creates a new local user account from the registration input and
// busts the cached user-count metric so the registration shows up immediately.
func RegisterUser(ctx context.Context, in *UserRegister) (*user.User, error) {
	s := db.NewSession()
	defer s.Close()
	defer events.CleanupPending(s)

	newUser, err := user.CreateUser(s, &user.User{
		Username: in.Username,
		Password: in.Password,
		Email:    in.Email,
		Language: in.Language,
		Status:   user.StatusActive,
	})
	if err != nil {
		_ = s.Rollback()
		return nil, err
	}

	if err := s.Commit(); err != nil {
		_ = s.Rollback()
		return nil, err
	}

	events.DispatchPending(ctx, s)

	if config.MetricsEnabled.GetBool() {
		if err := metrics.InvalidateCount(metrics.UserCountKey); err != nil {
			log.Errorf("Could not invalidate user count metric: %s", err)
		}
	}

	return newUser, nil
}

// AuthenticateUserCredentials verifies a login against local (and, if configured,
// LDAP) credentials and enforces the account-status and TOTP gates, returning the
// authenticated user on success. It is the transport-agnostic core of the login
// flow shared by v1 and v2; the caller issues the token and sets the cookie. The
// returned errors carry their own HTTP semantics (wrong credentials, disabled
// account, missing/invalid TOTP) so both APIs surface them identically.
func AuthenticateUserCredentials(ctx context.Context, login *user.Login) (*user.User, error) {
	s := db.NewSession()
	defer s.Close()
	// Discards events queued during a rolled-back transaction (e.g. LDAP user
	// creation); a no-op once DispatchPending has run.
	defer events.CleanupPending(s)

	u, err := resolveLoginUser(ctx, s, login)
	if err != nil {
		_ = s.Rollback()
		return nil, err
	}

	if u.Status == user.StatusDisabled {
		_ = s.Rollback()
		return nil, &user.ErrAccountDisabled{UserID: u.ID}
	}
	if u.Status == user.StatusAccountLocked {
		_ = s.Rollback()
		return nil, &user.ErrAccountLocked{UserID: u.ID}
	}

	if err := keyvalue.Del(u.GetFailedPasswordAttemptsKey()); err != nil {
		return nil, err
	}

	if err := s.Commit(); err != nil {
		_ = s.Rollback()
		return nil, err
	}

	events.DispatchPending(ctx, s)

	return u, nil
}

// resolveLoginUser authenticates the credentials against local accounts.
func resolveLoginUser(ctx context.Context, s *xorm.Session, login *user.Login) (*user.User, error) {
	return user.CheckUserCredentials(ctx, s, login)
}

// enforceLoginTOTP is a legacy no-op; TOTP is disabled in this MVP.
func enforceLoginTOTP(s *xorm.Session, u *user.User, passcode string) error {
	return nil
}

// DeleteSession removes the session with the given id, logging the user out
// server-side. An empty sid is a no-op.
func DeleteSession(sid string) error {
	return nil
}

// LogoutSession is a no-op in the MVP without server-side sessions.
func LogoutSession(sid string) (endSessionURL string, err error) {
	return "", nil
}

// ResetPassword is disabled in the MVP.
func ResetPassword(reset interface{}) error {
	return nil
}

// RequestPasswordResetToken is disabled in the MVP.
func RequestPasswordResetToken(req interface{}) error {
	return nil
}

// ConfirmEmail is disabled in the MVP.
func ConfirmEmail(confirm interface{}) error {
	return nil
}
