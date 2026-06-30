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

// TaskCollection is a struct used to hold filter details and not clutter the Task struct.
type TaskCollection struct {
	ProjectID int64 `param:"project" json:"-"`

	Search string `query:"s" json:"s" doc:"A search term to match tasks by their title."`

	SortBy []string `query:"sort_by" json:"sort_by" doc:"The fields to sort by."`

	OrderBy []string `query:"order_by" json:"order_by" doc:"The order for each sort_by field, either asc or desc."`

	web.CRUDable    `xorm:"-" json:"-"`
	web.Permissions `xorm:"-" json:"-"`
}

func getRelevantProjectsFromCollection(s *xorm.Session, a web.Auth, tf *TaskCollection) (projects []*Project, err error) {
	if tf.ProjectID == 0 {
		projects, _, _, err = getRawProjectsForUser(
			s,
			&projectOptions{
				user: &user.User{ID: a.GetID()},
				page: -1,
			},
		)
		return projects, err
	}

	project := &Project{ID: tf.ProjectID}
	canRead, _, err := project.CanRead(s, a)
	if err != nil {
		return nil, err
	}
	if !canRead {
		return nil, ErrUserDoesNotHaveAccessToProject{
			ProjectID: tf.ProjectID,
			UserID:    a.GetID(),
		}
	}

	return []*Project{{ID: tf.ProjectID}}, nil
}

// ReadAll gets all tasks for a project
func (tf *TaskCollection) ReadAll(s *xorm.Session, a web.Auth, search string, page int, perPage int) (result interface{}, resultCount int, totalItems int64, err error) {
	projects, err := getRelevantProjectsFromCollection(s, a, tf)
	if err != nil {
		return nil, 0, 0, err
	}

	tasks := []*Task{}
	projectIDs := make([]int64, len(projects))
	for i, p := range projects {
		projectIDs[i] = p.ID
	}

	query := s.In("project_id", projectIDs)
	if search != "" {
		query = query.And("title LIKE ?", "%"+search+"%")
	}

	totalItems, err = query.FindAndCount(&tasks)
	if err != nil {
		return nil, 0, 0, err
	}

	limit, start := getLimitFromPageIndex(page, perPage)
	if limit > 0 {
		if err = query.Limit(limit, start).Find(&tasks); err != nil {
			return nil, 0, 0, err
		}
	}

	return tasks, len(tasks), totalItems, nil
}
