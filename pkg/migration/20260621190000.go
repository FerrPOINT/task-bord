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

package migration

import (
	"fmt"
	"regexp"
	"strings"

	"src.techknowlogick.com/xormigrate"
	"xorm.io/xorm"
)

// projectIdentifier20260621190000 mirrors Project with a unique identifier column.
type projectIdentifier20260621190000 struct {
	ID         int64  `xorm:"bigint autoincr not null unique pk"`
	Title      string `xorm:"varchar(250) not null"`
	Identifier string `xorm:"varchar(10) null unique"`
}

func (projectIdentifier20260621190000) TableName() string {
	return "projects"
}

func init() {
	migrations = append(migrations, &xormigrate.Migration{
		ID:          "20260621190000",
		Description: "Add unique index on projects.identifier for task identifiers",
		Migrate: func(tx *xorm.Engine) error {
			// Existing projects may have empty identifiers, which would violate a
			// unique index. Generate a unique slug from the title for each one.
			projects := []*projectIdentifier20260621190000{}
			if err := tx.Where("identifier = '' OR identifier IS NULL").Find(&projects); err != nil {
				return err
			}

			for _, p := range projects {
				slug := generateProjectSlug(p.Title)
				candidate := slug
				for i := 1; ; i++ {
					exists, err := tx.Where("identifier = ?", candidate).Exist(&projectIdentifier20260621190000{})
					if err != nil {
						return err
					}
					if !exists {
						break
					}
					candidate = fmt.Sprintf("%s%d", slug, i)
					if len(candidate) > 10 {
						candidate = fmt.Sprintf("P%d", p.ID)
					}
				}
				_, err := tx.Exec("UPDATE projects SET identifier = ? WHERE id = ?", candidate, p.ID)
				if err != nil {
					return err
				}
			}

			return tx.Sync(projectIdentifier20260621190000{})
		},
		Rollback: func(tx *xorm.Engine) error {
			_, err := tx.Exec("DROP INDEX IF EXISTS UQE_projects_identifier")
			return err
		},
	})
}

// generateProjectSlug creates a 1–10 char uppercase identifier from a project title.
func generateProjectSlug(title string) string {
	// Keep only ASCII letters and digits, uppercase.
	re := regexp.MustCompile(`[^A-Z0-9]+`)
	clean := strings.ToUpper(re.ReplaceAllString(title, ""))
	if clean == "" {
		clean = "PROJECT"
	}
	if len(clean) > 10 {
		clean = clean[:10]
	}
	return clean
}
