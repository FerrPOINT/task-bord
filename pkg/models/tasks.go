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
	"strconv"
	"strings"
	"time"

	"github.com/FerrPOINT/task-bord/pkg/config"
	"github.com/FerrPOINT/task-bord/pkg/db"
	"github.com/FerrPOINT/task-bord/pkg/log"
	"github.com/FerrPOINT/task-bord/pkg/user"
	"github.com/FerrPOINT/task-bord/pkg/web"

	"xorm.io/builder"
	"xorm.io/xorm"
)

// Task represents a task in a project.
type Task struct {
	// The unique, numeric id of this task.
	ID int64 `xorm:"bigint autoincr not null unique pk" json:"id" readOnly:"true" doc:"The unique, numeric id of this task."`
	// ProjectTaskParam holds the raw {projecttask} URL segment for v1 task routes.
	ProjectTaskParam string `xorm:"-" param:"projecttask"`
	// The task title.
	Title string `xorm:"TEXT not null" json:"title" valid:"minstringlength(1)" minLength:"1" doc:"The task title."`
	// The task description.
	Description string `xorm:"longtext null" json:"description"`
	// Whether a task is done or not.
	Done bool `xorm:"INDEX null" json:"done"`
	// The time when a task was marked as done.
	DoneAt time.Time `xorm:"INDEX null 'done_at'" json:"done_at" readOnly:"true"`
	// The time when the task is due.
	DueDate time.Time `xorm:"DATETIME INDEX null 'due_date'" json:"due_date"`

	// The project this task belongs to.
	ProjectID int64 `xorm:"bigint INDEX not null unique(tasks_project_index)" json:"project_id" param:"project" doc:"The id of the project this task belongs to."`

	// An array of users who are assigned to this task (read-only here).
	Assignees []*user.User `xorm:"-" json:"assignees" readOnly:"true"`
	// An array of labels associated with this task (read-only here).
	Labels []*Label `xorm:"-" json:"labels" readOnly:"true"`

	// The task identifier, based on the project identifier and the task's index.
	Identifier string `xorm:"-" json:"identifier" readOnly:"true"`
	// The task index, calculated per project.
	Index int64 `xorm:"bigint not null default 0 unique(tasks_project_index)" json:"index" param:"index" readOnly:"true"`

	// A timestamp when this task was created.
	Created time.Time `xorm:"created not null" json:"created" readOnly:"true"`
	// A timestamp when this task was last updated.
	Updated time.Time `xorm:"updated not null" json:"updated" readOnly:"true"`

	// All comments of this task.
	Comments []*TaskComment `xorm:"-" json:"comments,omitempty" readOnly:"true"`
	// Comment count of this task.
	CommentCount *int64 `xorm:"-" json:"comment_count,omitempty" readOnly:"true"`

	// The user who initially created the task.
	CreatedBy   *user.User `xorm:"-" json:"created_by" valid:"-" readOnly:"true"`
	CreatedByID int64      `xorm:"bigint not null" json:"-"`

	web.CRUDable    `xorm:"-" json:"-"`
	web.Permissions `xorm:"-" json:"-"`
}

// TaskWithComments is a task with its comments included.
type TaskWithComments struct {
	Task
	Comments []*TaskComment `xorm:"-" json:"comments"`
}

// TableName returns the table name for tasks.
func (t *Task) TableName() string {
	return "tasks"
}

// GetIdentifier returns the textual identifier for this task.
func (t *Task) GetIdentifier() string {
	if t.Identifier != "" {
		return t.Identifier
	}

	project, err := GetProjectSimpleByID(db.NewSession(), t.ProjectID)
	if err == nil && project != nil && project.Identifier != "" {
		return project.Identifier + "-" + strconv.FormatInt(t.Index, 10)
	}

	return "#" + strconv.FormatInt(t.Index, 10)
}

// GetFrontendURL returns the public URL for this task.
func (t *Task) GetFrontendURL() string {
	return config.ServicePublicURL.GetString() + "tasks/" + strconv.FormatInt(t.ID, 10)
}

// CanCreate checks if the user can create a task in the project.
func (t *Task) CanCreate(s *xorm.Session, a web.Auth) (bool, error) {
	project := &Project{ID: t.ProjectID}
	return project.CanUpdate(s, a)
}

// Create creates a new task in a project.
func (t *Task) Create(s *xorm.Session, a web.Auth) error {
	if t.ProjectID == 0 {
		return ErrProjectCannotBeEmpty{}
	}

	project := &Project{ID: t.ProjectID}
	if _, err := s.Get(project); err != nil || project.ID == 0 {
	return ErrProjectDoesNotExist{ID: t.ProjectID}
}

	t.CreatedByID = a.GetID()

	// Calculate the next index for this project
	var maxIndex int64
	_, err := s.SQL("SELECT COALESCE(MAX(`index`), 0) FROM tasks WHERE project_id = ?", t.ProjectID).Get(&maxIndex)
	if err != nil {
		return err
	}
	t.Index = maxIndex + 1
	t.Identifier = project.Identifier + "-" + strconv.FormatInt(t.Index, 10)

	_, err = s.Insert(t)
	if err != nil {
		return err
	}

	return updateTaskProjectUpdated(s, t.ID)
}

// ReadOne reads a single task by ID or by ProjectID+Index.
func (t *Task) ReadOne(s *xorm.Session, a web.Auth) error {
	if t.ID != 0 {
		if _, err := s.ID(t.ID).Get(t); err != nil {
			return err
		}
		if t.ID == 0 {
			return ErrTaskDoesNotExist{ID: t.ID}
		}
	} else if t.ProjectID != 0 && t.Index != 0 {
		if _, err := s.Where("project_id = ? AND `index` = ?", t.ProjectID, t.Index).Get(t); err != nil {
			return err
		}
		if t.ID == 0 {
			return ErrTaskDoesNotExist{}
		}
	} else {
		return ErrTaskDoesNotExist{}
	}

	t.Identifier = t.GetIdentifier()

	// Load assignees
	if err := t.loadAssignees(s); err != nil {
		return err
	}
	// Load labels
	if err := t.loadLabels(s); err != nil {
		return err
	}

	return nil
}

// ReadAll is a dummy implementation required by the CRUDable interface.
func (t *Task) ReadAll(_ *xorm.Session, _ web.Auth, _ string, _ int, _ int) (result interface{}, resultCount int, totalItems int64, err error) {
	return nil, 0, 0, nil
}

// Update updates a task.
func (t *Task) Update(s *xorm.Session, a web.Auth) error {
	if t.ID == 0 {
		return ErrTaskDoesNotExist{}
	}

	existing := &Task{ID: t.ID}
	if _, err := s.Get(existing); err != nil {
		return err
	}
	if existing.ID == 0 {
		return ErrTaskDoesNotExist{ID: t.ID}
	}

	// Copy immutable fields from existing
	t.ProjectID = existing.ProjectID
	t.Index = existing.Index
	t.Created = existing.Created
	t.CreatedByID = existing.CreatedByID
	t.Identifier = existing.GetIdentifier()

	if _, err := s.ID(t.ID).Update(t); err != nil {
		return err
	}

	return updateTaskProjectUpdated(s, t.ID)
}

// Delete deletes a task.
func (t *Task) Delete(s *xorm.Session, a web.Auth) error {
	if t.ID == 0 {
		return ErrTaskDoesNotExist{}
	}

	existing := &Task{ID: t.ID}
	if _, err := s.Get(existing); err != nil {
		return err
	}
	if existing.ID == 0 {
		return ErrTaskDoesNotExist{ID: t.ID}
	}

	if _, err := s.ID(t.ID).Delete(&Task{}); err != nil {
		return err
	}

	return updateTaskProjectUpdated(s, t.ID)
}

// GetTaskByProjectAndIndex returns a task by its project and per-project index.
func GetTaskByProjectAndIndex(s *xorm.Session, projectID, index int64) (*Task, error) {
	t := &Task{ProjectID: projectID, Index: index}
	if _, err := s.Where("project_id = ? AND `index` = ?", projectID, index).Get(t); err != nil {
		return nil, err
	}
	if t.ID == 0 {
		return nil, ErrTaskDoesNotExist{}
	}
	return t, nil
}

// ResolveProjectTaskParam populates t.ID from the raw {projecttask} URL segment.
// It accepts either a numeric id or a textual identifier (e.g. "PROJ-123").
func (t *Task) ResolveProjectTaskParam(s *xorm.Session) error {
	if t.ProjectTaskParam == "" {
		return nil
	}
	if t.ID != 0 {
		return nil
	}

	// Numeric id
	if id, err := strconv.ParseInt(t.ProjectTaskParam, 10, 64); err == nil {
		t.ID = id
		return nil
	}

	parts := strings.SplitN(t.ProjectTaskParam, "-", 2)
	if len(parts) != 2 {
		return ErrTaskDoesNotExist{}
	}
	project, err := GetProjectSimpleByIdentifier(s, parts[0])
	if err != nil {
		return err
	}
	index, err := strconv.ParseInt(parts[1], 10, 64)
	if err != nil {
		return ErrTaskDoesNotExist{}
	}
	resolved, err := GetTaskByProjectAndIndex(s, project.ID, index)
	if err != nil {
		return err
	}
	t.ID = resolved.ID
	return nil
}

// GetTasksByIDs returns tasks by their IDs, scoped to a project.
func GetTasksByIDs(s *xorm.Session, projectID int64, ids []int64) (tasks []*Task, err error) {
	if len(ids) == 0 {
		return nil, nil
	}
	err = s.Where("project_id = ?", projectID).In("id", ids).Find(&tasks)
	return tasks, err
}

func (t *Task) loadAssignees(s *xorm.Session) error {
	assignees, err := getRawTaskAssigneesForTasks(s, []int64{t.ID})
	if err != nil {
		return err
	}
	t.Assignees = make([]*user.User, 0, len(assignees))
	for i := range assignees {
		t.Assignees = append(t.Assignees, &assignees[i].User)
	}
	return nil
}

func (t *Task) loadLabels(s *xorm.Session) error {
	labels, _, _, err := GetLabelsByTaskIDs(s, &LabelByTaskIDsOptions{
		TaskIDs:             []int64{t.ID},
		GroupByLabelIDsOnly: true,
	})
	if err != nil {
		return err
	}
	t.Labels = make([]*Label, 0, len(labels))
	for _, l := range labels {
		t.Labels = append(t.Labels, &l.Label)
	}
	return nil
}

// GetTaskSimple returns a task by id with minimal fields populated.
func GetTaskSimple(s *xorm.Session, taskID int64) (*Task, error) {
	t := &Task{ID: taskID}
	if _, err := s.Get(t); err != nil {
		return nil, err
	}
	if t.ID == 0 {
		return nil, ErrTaskDoesNotExist{}
	}
	return t, nil
}

// GetTaskByIDSimple is an alias for GetTaskSimple.
func GetTaskByIDSimple(s *xorm.Session, taskID int64) (*Task, error) {
	return GetTaskSimple(s, taskID)
}

// triggerTaskUpdatedEventForTaskID is a no-op placeholder in MVP.
func triggerTaskUpdatedEventForTaskID(s *xorm.Session, auth web.Auth, taskID int64) error {
	return nil
}

// getTaskIndexFromSearchString extracts a numeric index from a #123-style string.
func getTaskIndexFromSearchString(s string) (index int64) {
	if !strings.HasPrefix(s, "#") {
		return 0
	}
	index, _ = strconv.ParseInt(s[1:], 10, 64)
	return index
}

// updateTaskProjectUpdated updates the project's updated timestamp for a task.
func updateTaskProjectUpdated(s *xorm.Session, taskID int64) error {
	if s == nil {
		s = db.NewSession()
		defer s.Close()
	}

	_, err := s.Exec(builder.Update(builder.Eq{"updated": time.Now()}).
		From("projects").
		Where(builder.In("id", builder.Select("project_id").From("tasks").Where(builder.Eq{"id": taskID}))),
	)
	if err != nil {
		log.Debugf("Could not update project updated date: %s", err)
	}
	return err
}
