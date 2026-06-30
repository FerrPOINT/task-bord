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
	"strconv"

	"github.com/FerrPOINT/task-bord/pkg/db"
	"github.com/FerrPOINT/task-bord/pkg/models"
	"github.com/FerrPOINT/task-bord/pkg/web/handler"

	"github.com/danielgtaylor/huma/v2"
	"github.com/danielgtaylor/huma/v2/conditional"
)

// RegisterTaskRoutes wires Task CRUD onto the Huma API. The list lives on
// TaskCollection, not here.
func RegisterTaskRoutes(api huma.API) {
	tags := []string{"tasks"}

	Register(api, huma.Operation{
		OperationID: "tasks-read",
		Summary:     "Get a task",
		Description: "Returns a single task by its numeric id. Sends an ETag; pass it as If-None-Match on a later read to get a 304 Not Modified.",
		Method:      "GET",
		Path:        "/tasks/{projecttask}",
		Tags:        tags,
	}, tasksRead)

	Register(api, huma.Operation{
		OperationID: "tasks-read-by-index",
		Summary:     "Get a task by its project index",
		Description: "Returns a single task addressed by its per-project index. The {project} segment accepts either a numeric project id or a textual project identifier; a value made solely of digits is always treated as an id.",
		Method:      "GET",
		Path:        "/projects/{project}/tasks/by-index/{index}",
		Tags:        tags,
	}, tasksReadByIndex)

	Register(api, huma.Operation{
		OperationID: "tasks-create",
		Summary:     "Create a task",
		Description: "Creates a task in the project from the URL. The authenticated user needs write access to that project and becomes the task's creator.",
		Method:      "POST",
		Path:        "/projects/{project}/tasks",
		Tags:        tags,
	}, tasksCreate)

	Register(api, huma.Operation{
		OperationID: "tasks-update",
		Summary:     "Update a task",
		Description: "Replaces all of a task's fields; requires write access. Use PATCH for a partial update.",
		Method:      "PUT",
		Path:        "/tasks/{projecttask}",
		Tags:        tags,
	}, tasksUpdate)

	Register(api, huma.Operation{
		OperationID: "tasks-delete",
		Summary:     "Delete a task",
		Description: "Deletes a task. Requires write access to its project.",
		Method:      "DELETE",
		Path:        "/tasks/{projecttask}",
		Tags:        tags,
	}, tasksDelete)
}

func init() { AddRouteRegistrar(RegisterTaskRoutes) }

type taskReadOneBody struct {
	models.Task
	MaxPermission models.Permission `json:"max_permission" readOnly:"true" doc:"The maximum permission the requesting user has on this task (0=read, 1=read/write, 2=admin)."`
}

func tasksRead(ctx context.Context, in *struct {
	ID int64 `path:"projecttask" doc:"The numeric id of the task."`
	conditional.Params
}) (*singleReadBody[taskReadOneBody], error) {
	a, err := authFromCtx(ctx)
	if err != nil {
		return nil, err
	}
	task := &models.Task{ID: in.ID}
	maxPermission, err := handler.DoReadOne(ctx, task, a)
	if err != nil {
		return nil, translateDomainError(err)
	}
	body := &taskReadOneBody{Task: *task, MaxPermission: models.Permission(maxPermission)}
	return conditionalReadResponse(&in.Params, body, task.Updated, maxPermission)
}

func tasksReadByIndex(ctx context.Context, in *struct {
	Project string `path:"project" doc:"A numeric project id or a textual project identifier."`
	Index   int64  `path:"index" doc:"The per-project task index."`
	conditional.Params
}) (*singleReadBody[taskReadOneBody], error) {
	a, err := authFromCtx(ctx)
	if err != nil {
		return nil, err
	}
	projectID, err := resolveProjectIdentifier(in.Project)
	if err != nil {
		return nil, err
	}

	task := &models.Task{ProjectID: projectID, Index: in.Index}
	maxPermission, err := handler.DoReadOne(ctx, task, a)
	if err != nil {
		return nil, translateDomainError(err)
	}
	body := &taskReadOneBody{Task: *task, MaxPermission: models.Permission(maxPermission)}
	return conditionalReadResponse(&in.Params, body, task.Updated, maxPermission)
}

func tasksCreate(ctx context.Context, in *struct {
	Project int64 `path:"project" doc:"The numeric id of the project to create the task in."`
	Body    models.Task
}) (*singleBody[models.Task], error) {
	a, err := authFromCtx(ctx)
	if err != nil {
		return nil, err
	}
	task := &in.Body
	task.ProjectID = in.Project // URL wins over body
	if err := handler.DoCreate(ctx, task, a); err != nil {
		return nil, translateDomainError(err)
	}
	return &singleBody[models.Task]{Body: task}, nil
}

func tasksUpdate(ctx context.Context, in *struct {
	ID   int64 `path:"projecttask"`
	Body taskReadOneBody
}) (*singleBody[models.Task], error) {
	a, err := authFromCtx(ctx)
	if err != nil {
		return nil, err
	}
	task := &in.Body.Task
	task.ID = in.ID // URL wins over body
	if err := handler.DoUpdate(ctx, task, a); err != nil {
		return nil, translateDomainError(err)
	}
	return &singleBody[models.Task]{Body: task}, nil
}

func tasksDelete(ctx context.Context, in *struct {
	ID int64 `path:"projecttask"`
}) (*emptyBody, error) {
	a, err := authFromCtx(ctx)
	if err != nil {
		return nil, err
	}
	if err := handler.DoDelete(ctx, &models.Task{ID: in.ID}, a); err != nil {
		return nil, translateDomainError(err)
	}
	return &emptyBody{}, nil
}

func resolveProjectIdentifier(raw string) (int64, error) {
	if id, err := strconv.ParseInt(raw, 10, 64); err == nil {
		return id, nil
	}
	s := db.NewSession()
	defer s.Close()
	project, err := models.GetProjectSimpleByIdentifier(s, raw)
	if err != nil {
		return 0, translateDomainError(err)
	}
	return project.ID, nil
}
