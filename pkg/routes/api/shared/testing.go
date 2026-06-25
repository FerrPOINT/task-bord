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
	"github.com/FerrPOINT/task-bord/pkg/db"
	"github.com/FerrPOINT/task-bord/pkg/events"
)

// dependentTestingTables lists tables that reference a reset table by ID and
// must be truncated alongside it. Without foreign key cascades, stale rows
// would persist and pollute subsequent tests that reuse the same
// auto-increment IDs.
var dependentTestingTables = map[string][]string{
	"users": {"notifications"},
}

// ReplaceTableContents resets a single table to the provided rows for the e2e
// testing endpoint and returns the table's resulting contents. When truncate is
// true the table (and any dependent tables) is emptied first; otherwise the rows
// are restored on top of existing data. Callers must already have verified the
// testing token.
func ReplaceTableContents(table string, content []map[string]interface{}, truncate bool) ([]map[string]interface{}, error) {
	// Wait for all async event handlers from the previous test to complete
	// before modifying the database. Without this, handlers hold SQLite
	// connections and starve this request's truncate/insert operations.
	events.WaitForPendingHandlers()

	var err error
	if truncate {
		for _, dep := range dependentTestingTables[table] {
			if err = db.RestoreAndTruncate(dep, nil); err != nil {
				return nil, err
			}
		}
		err = db.RestoreAndTruncate(table, content)
	} else {
		err = db.Restore(table, content)
	}
	if err != nil {
		return nil, err
	}


	s := db.NewSession()
	defer s.Close()
	data := []map[string]interface{}{}
	if err := s.Table(table).Find(&data); err != nil {
		return nil, err
	}
	return data, nil
}

// TruncateAllTestingTables empties every taskboard table for the e2e testing
// endpoint. Callers must already have verified the testing token.
func TruncateAllTestingTables() error {
	events.WaitForPendingHandlers()

	if err := db.TruncateAllTables(); err != nil {
		return err
	}

	return nil
}
