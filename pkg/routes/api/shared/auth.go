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
	"github.com/FerrPOINT/task-bord/pkg/models"
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
// The caller is responsible for the registration-enabled gate and input
// validation; both v1 and v2 share this body.
func RegisterUser(ctx context.Context, in *UserRegister) (*user.User, error) {
	s := db.NewSession()
	defer s.Close()
	// Discards events queued during a rolled-back transaction; a no-op once
	// DispatchPending has run.
	defer events.CleanupPending(s)

	newUser, err := models.RegisterUser(s, &user.User{
		Username: in.Username,
		Password: in.Password,
		Email:    in.Email,
		Language: in.Language,
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

	// Bust the cached user count so the new registration shows up in metrics
	// immediately instead of after the regular cache expiry.
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
// server-side. An empty sid is a no-op (the token carried no session, e.g. an
// API token or a link share), matching v1. Shared by v1 and v2; the caller is
// responsible for clearing the refresh cookie.
func DeleteSession(sid string) error {
	_, err := LogoutSession(sid)
	return err
}

// LogoutSession deletes the session and returns its OIDC RP-Initiated Logout URL
// for the frontend to redirect to (empty for non-OIDC sessions or when no logout
// endpoint is configured). An empty sid is a no-op. The caller clears the refresh
// cookie.
func LogoutSession(sid string) (endSessionURL string, err error) {
	if sid == "" {
		return "", nil
	}

	s := db.NewSession()
	defer s.Close()

	// Read before deleting so the stored id_token survives for the logout URL.
	// A missing session just means there is nothing to log out.
	// No external OIDC logout in MVP; just delete the local session.
	if _, err := s.Where("id = ?", sid).Delete(&models.Session{}); err != nil {
		_ = s.Rollback()
		return "", err
	}

	if err := s.Commit(); err != nil {
		_ = s.Rollback()
		return "", err
	}

	return endSessionURL, nil
}

// ResetPassword resets a user's password from a previously issued reset token
// and invalidates all of that user's sessions, so a leaked password cannot be
// used after a reset. Shared by v1 and v2.
func ResetPassword(reset *user.PasswordReset) error {
	s := db.NewSession()
	defer s.Close()

	userID, err := user.ResetPassword(s, reset)
	if err != nil {
		_ = s.Rollback()
		return err
	}

	if err := models.DeleteAllUserSessions(s, userID); err != nil {
		_ = s.Rollback()
		return err
	}

	return s.Commit()
}

// RequestPasswordResetToken issues a password-reset token for the account with
// the given email and sends it via email. Shared by v1 and v2.
func RequestPasswordResetToken(req *user.PasswordTokenRequest) error {
	s := db.NewSession()
	defer s.Close()

	if err := user.RequestUserPasswordResetTokenByEmail(s, req); err != nil {
		_ = s.Rollback()
		return err
	}

	return s.Commit()
}

// ConfirmEmail confirms a newly registered user's email from the token sent to
// them. Shared by v1 and v2.
func ConfirmEmail(confirm *user.EmailConfirm) error {
	s := db.NewSession()
	defer s.Close()

	if err := user.ConfirmEmail(s, confirm); err != nil {
		_ = s.Rollback()
		return err
	}

	return s.Commit()
}
