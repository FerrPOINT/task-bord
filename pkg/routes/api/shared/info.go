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
	"github.com/FerrPOINT/task-bord/pkg/config"
	"github.com/FerrPOINT/task-bord/pkg/version"
)

// TaskBoardInfos holds public information about this taskboard instance.
type TaskBoardInfos struct {
	Version                string    `json:"version" doc:"The taskboard version this instance runs."`
	FrontendURL            string    `json:"frontend_url" doc:"The publicly configured frontend URL of this instance."`
	Motd                   string    `json:"motd" doc:"The message of the day, shown to all users."`
	MaxFileSize            string    `json:"max_file_size" doc:"The maximum allowed upload size, as a human-readable string (e.g. 20MB)."`
	MaxItemsPerPage        int       `json:"max_items_per_page" doc:"The maximum number of items a paginated endpoint returns per page."`
	TaskAttachmentsEnabled bool      `json:"task_attachments_enabled" doc:"Whether task attachments are enabled."`
	Legal                  LegalInfo `json:"legal" doc:"Links to the instance's legal documents."`
	AuthInfo               AuthInfo  `json:"auth" doc:"The authentication methods enabled on this instance."`
	EmailRemindersEnabled  bool      `json:"email_reminders_enabled" doc:"Whether email reminders are enabled."`
	UserDeletionEnabled    bool      `json:"user_deletion_enabled" doc:"Whether users may delete their own account."`
	TaskCommentsEnabled    bool      `json:"task_comments_enabled" doc:"Whether task comments are enabled."`
	DemoModeEnabled        bool      `json:"demo_mode_enabled" doc:"Whether this instance runs in demo mode (data is periodically reset)."`
	PublicTeamsEnabled     bool      `json:"public_teams_enabled" doc:"Whether public teams are enabled."`
	AllowIconChanges       bool      `json:"allow_icon_changes" doc:"Whether users may change project icons."`
	// ConcurrentWrites reports whether the configured database can handle concurrent writes. It is false on SQLite, where overlapping write transactions deadlock, so clients should serialize batched writes instead of firing them in parallel.
	ConcurrentWrites bool `json:"concurrent_writes" doc:"Whether the configured database supports concurrent writes. False on SQLite; clients should serialize batched writes when this is false."`
}

// AuthInfo describes the authentication methods enabled on this instance.
type AuthInfo struct {
	Local         LocalAuthInfo  `json:"local"`
}

// LocalAuthInfo describes the local (username/password) authentication method.
type LocalAuthInfo struct {
	Enabled             bool `json:"enabled"`
	RegistrationEnabled bool `json:"registration_enabled"`
}


// LegalInfo holds links to the instance's legal documents.
type LegalInfo struct {
	ImprintURL       string `json:"imprint_url"`
	PrivacyPolicyURL string `json:"privacy_policy_url"`
}

// BuildInfo assembles the public instance information returned by GET /info on
// both API versions.
func BuildInfo() TaskBoardInfos {
	info := TaskBoardInfos{
		Version:                version.Version,
		FrontendURL:            config.ServicePublicURL.GetString(),
		Motd:                   config.ServiceMotd.GetString(),
		MaxFileSize:            config.FilesMaxSize.GetString(),
		MaxItemsPerPage:        config.ServiceMaxItemsPerPage.GetInt(),
		TaskAttachmentsEnabled: config.ServiceEnableTaskAttachments.GetBool(),
		EmailRemindersEnabled:  config.ServiceEnableEmailReminders.GetBool(),
		UserDeletionEnabled:    config.ServiceEnableUserDeletion.GetBool(),
		TaskCommentsEnabled:    config.ServiceEnableTaskComments.GetBool(),
		DemoModeEnabled:        config.ServiceDemoMode.GetBool(),
		PublicTeamsEnabled:     config.ServiceEnablePublicTeams.GetBool(),
		AllowIconChanges:       config.ServiceAllowIconChanges.GetBool(),
		ConcurrentWrites:       config.DatabaseType.GetString() != "sqlite",
		Legal: LegalInfo{
			ImprintURL:       "",
			PrivacyPolicyURL: "",
		},
		AuthInfo: AuthInfo{
			Local: LocalAuthInfo{
				Enabled:             config.AuthLocalEnabled.GetBool(),
				RegistrationEnabled: config.AuthLocalEnabled.GetBool() && config.ServiceEnableRegistration.GetBool(),
			},
		},
	}


	if config.BackgroundsEnabled.GetBool() {
		_ = config.BackgroundsUploadEnabled.GetBool()
	}

	return info
}