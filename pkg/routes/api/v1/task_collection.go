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

package apiv1

import (
	"context"
	"fmt"
	"net/http"

	"github.com/FerrPOINT/task-bord/pkg/models"
	"github.com/FerrPOINT/task-bord/pkg/web/handler"

	"github.com/danielgtaylor/huma/v2"
)

type taskListBody struct {
	Body Paginated[*models.Task]
}

// TaskListQueryParams is the shared filter/sort/search query block.
type TaskListQueryParams struct {
	ListParams
	SortBy  []string `query:"sort_by,explode" doc:"Fields to sort by."`
	OrderBy []string `query:"order_by,explode" doc:"Sort order per sort_by field, asc or desc."`
}

type taskListAllInput struct {
	TaskListQueryParams
}

type taskListProjectInput struct {
	ProjectID int64 `path:"project" doc:"The numeric id of the project."`
	TaskListQueryParams
}

type taskListFilters struct {
	Q       string
	SortBy  []string
	OrderBy []string
}

func (in taskListAllInput) filters() taskListFilters {
	return taskListFilters{in.Q, in.SortBy, in.OrderBy}
}

func (in taskListProjectInput) filters() taskListFilters {
	return taskListFilters{in.Q, in.SortBy, in.OrderBy}
}

func (f taskListFilters) collection(projectID int64) *models.TaskCollection {
	return &models.TaskCollection{
		ProjectID: projectID,
		SortBy:    f.SortBy,
		OrderBy:   f.OrderBy,
	}
}

func RegisterTaskCollectionRoutes(api huma.API) {
	tags := []string{"tasks"}

	Register(api, huma.Operation{
		OperationID: "tasks-list",
		Summary:     "List tasks across all projects",
		Description: "Returns the tasks the authenticated user can see across every project they have access to, paginated and flat.",
		Method:      http.MethodGet,
		Path:        "/tasks",
		Tags:        tags,
	}, tasksListAll)

	Register(api, huma.Operation{
		OperationID: "project-tasks-list",
		Summary:     "List tasks in a project",
		Description: "Returns the tasks in a project, paginated and flat. Requires read access to the project.",
		Method:      http.MethodGet,
		Path:        "/projects/{project}/tasks",
		Tags:        tags,
	}, projectTasksList)
}

func init() { AddRouteRegistrar(RegisterTaskCollectionRoutes) }

func tasksListAll(ctx context.Context, in *taskListAllInput) (*taskListBody, error) {
	return readFlatTasks(ctx, in.filters(), in.Page, in.PerPage, 0)
}

func projectTasksList(ctx context.Context, in *taskListProjectInput) (*taskListBody, error) {
	return readFlatTasks(ctx, in.filters(), in.Page, in.PerPage, in.ProjectID)
}

func readFlatTasks(ctx context.Context, f taskListFilters, page, perPage int, projectID int64) (*taskListBody, error) {
	a, err := authFromCtx(ctx)
	if err != nil {
		return nil, err
	}
	tc := f.collection(projectID)
	result, _, total, err := handler.DoReadAll(ctx, tc, a, f.Q, page, perPage)
	if err != nil {
		return nil, translateDomainError(err)
	}
	tasks, ok := result.([]*models.Task)
	if !ok {
		return nil, fmt.Errorf("taskCollection.ReadAll returned unexpected type %T (expected []*models.Task)", result)
	}
	return &taskListBody{Body: NewPaginated(tasks, total, page, perPage)}, nil
}
