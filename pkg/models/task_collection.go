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

// TaskCollection is a struct used to hold filter details and not clutter the Task struct with information not related to actual tasks.
type TaskCollection struct {
	ProjectID int64 `param:"project" json:"-"`

	Search string `query:"s" json:"s" doc:"A search term to match tasks by their title."`

	// The query parameter to sort by. This is for ex. done, priority, etc.
	SortBy []string `query:"sort_by" json:"sort_by" doc:"The fields to sort by, for example done or priority."`
	// The query parameter to order the items by. This can be either asc or desc, with asc being the default.
	OrderBy []string `query:"order_by" json:"order_by" doc:"The order for each sort_by field, either asc or desc. Defaults to asc."`

	// The filter query to match tasks by.
	Filter string `query:"filter" json:"filter" doc:"The filter query to match tasks by."`
	// The time zone which should be used for date match (statements like "now" resolve to different actual times)
	FilterTimezone string `query:"filter_timezone" json:"-"`

	// If set to true, the result will also include null values
	FilterIncludeNulls bool `query:"filter_include_nulls" json:"filter_include_nulls" doc:"If true, the result also includes tasks whose filtered field is null."`

	web.CRUDable    `xorm:"-" json:"-"`
	web.Permissions `xorm:"-" json:"-"`
}

func getTaskFilterOptsFromCollection(tf *TaskCollection) (opts *taskSearchOptions, err error) {
	var sort = make([]*sortParam, 0, len(tf.SortBy))
	for i, s := range tf.SortBy {
		param := &sortParam{
			sortBy:  s,
			orderBy: orderAscending,
		}
		if len(tf.OrderBy) > i {
			param.orderBy = getSortOrderFromString(tf.OrderBy[i])
		}
		if err := param.validate(); err != nil {
			return nil, err
		}
		sort = append(sort, param)
	}

	opts = &taskSearchOptions{
		sortby:             sort,
		filterIncludeNulls: tf.FilterIncludeNulls,
		filter:             tf.Filter,
		filterTimezone:     tf.FilterTimezone,
	}

	opts.parsedFilters, err = getTaskFiltersFromFilterString(tf.Filter, tf.FilterTimezone)
	return opts, err
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

	// Check the project exists and the user has access on it
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

// ReadAll gets all tasks for a collection
// @Summary Get tasks in a project
// @Description Returns all tasks for the selected project.
// @tags task
// @Accept json
// @Produce json
// @Param id path int true "The project ID."
// @Param page query int false "The page number. Used for pagination. If not provided, the first page of results is returned."
// @Param per_page query int false "The maximum number of items per page. Note this parameter is limited by the configured maximum of items per page."
// @Param s query string false "Search tasks by task text."
// @Param sort_by query string false "The sorting parameter."
// @Param order_by query string false "The ordering parameter."
// @Param filter query string false "The filter query to match tasks by."
// @Security JWTKeyAuth
// @Success 200 {array} models.Task "The tasks"
// @Failure 500 {object} models.Message "Internal error"
// @Router /projects/{id}/tasks [get]
func (tf *TaskCollection) ReadAll(s *xorm.Session, a web.Auth, search string, page int, perPage int) (result interface{}, resultCount int, totalItems int64, err error) {
	opts, err := getTaskFilterOptsFromCollection(tf)
	if err != nil {
		return nil, 0, 0, err
	}

	opts.search = search
	opts.page = page
	opts.perPage = perPage

	projects, err := getRelevantProjectsFromCollection(s, a, tf)
	if err != nil {
		return nil, 0, 0, err
	}

	return getTasksForProjects(s, projects, a, opts)
}
