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

package user

import (
	"github.com/FerrPOINT/task-bord/pkg/config"
	"github.com/FerrPOINT/task-bord/pkg/notifications"
	"xorm.io/xorm"
)

// PasswordReset holds the data to reset a password
type PasswordReset struct {
	// The previously issued reset token.
	Token string `json:"token"`
	// The new password for this user.
	NewPassword string `json:"new_password" valid:"bcrypt_password" minLength:"1" maxLength:"72"`
}

// ResetPassword resets a users password. It returns the ID of the user whose
// password was reset so callers can perform additional cleanup (e.g. session
// invalidation).
func ResetPassword(s *xorm.Session, reset *PasswordReset) (userID int64, err error) {

	// Check if the password is not empty
	if reset.NewPassword == "" {
		return 0, ErrNoUsernamePassword{}
	}

	if reset.Token == "" {
		return 0, ErrNoPasswordResetToken{}
	}

	// Token validation was removed along with the token table; in this MVP the
	// reset flow is disabled. Keep the endpoint returning an invalid token error.
	return 0, ErrInvalidPasswordResetToken{Token: reset.Token}
}

// PasswordTokenRequest defines the request format for password reset resqest
type PasswordTokenRequest struct {
	Email string `json:"email" valid:"email,length(0|250)" maxLength:"250"`
}

// RequestUserPasswordResetTokenByEmail is disabled in this MVP.
func RequestUserPasswordResetTokenByEmail(s *xorm.Session, tr *PasswordTokenRequest) (err error) {
	if tr.Email == "" {
		return ErrNoUsernamePassword{}
	}

	// Check if the user exists
	user, err := GetUserWithEmail(s, &User{Email: tr.Email})
	if err != nil && !IsErrAccountLocked(err) {
		return err
	}

	if !config.MailerEnabled.GetBool() {
		return nil
	}

	// Send a notification without a token; the flow is intentionally simplified.
	n := &ResetPasswordNotification{
		User: user,
	}

	return notifications.Notify(user, n, s)
}

// RequestUserPasswordResetToken is disabled in this MVP.
func RequestUserPasswordResetToken(s *xorm.Session, user *User) (err error) {
	if !config.MailerEnabled.GetBool() {
		return nil
	}

	n := &ResetPasswordNotification{
		User: user,
	}

	return notifications.Notify(user, n, s)
}
