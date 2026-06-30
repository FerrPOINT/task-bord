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
	_ "github.com/FerrPOINT/task-bord/pkg/config" // To trigger its init() which initializes the config
	"github.com/FerrPOINT/task-bord/pkg/db"
	"github.com/FerrPOINT/task-bord/pkg/log"
	"github.com/FerrPOINT/task-bord/pkg/mail"
	"github.com/FerrPOINT/task-bord/pkg/notifications"
)

// SetupTests takes care of seting up the db, fixtures etc.
// This is an extra function to be able to call the fixtures setup from the web tests.
func SetupTests() {
	var err error
	x, err = db.CreateTestEngine()
	if err != nil {
		log.Fatal(err)
	}

	tables := []interface{}{}
	tables = append(tables, GetTables()...)
	tables = append(tables, notifications.GetTables()...)

	err = x.Sync2(tables...)
	if err != nil {
		log.Fatal(err)
	}

	err = db.CreateParadeDBIndexes()
	if err != nil {
		log.Fatal(err)
	}

	err = db.InitTestFixtures(
		"files",
		"label_tasks",
		"labels",
		"projects",
		"task_assignees",
		"task_attachments",
		"task_comments",
		"tasks",
		"users",
		"users_projects",
	)
	if err != nil {
		log.Fatal(err)
	}

	// Start the pseudo mail queue
	mail.StartMailDaemon()
}
