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
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/FerrPOINT/task-bord/pkg/events"
	"github.com/FerrPOINT/task-bord/pkg/user"
	"github.com/FerrPOINT/task-bord/pkg/utils"
	"github.com/FerrPOINT/task-bord/pkg/web"

	"xorm.io/builder"
	"xorm.io/xorm"
)

// Project represents a project of tasks
type Project struct {
	// The unique, numeric id of this project.
	ID int64 `xorm:"bigint autoincr not null unique pk" json:"id" param:"project" readOnly:"true" doc:"The unique, numeric id of this project."`
	// The title of the project. You'll see this in the overview.
	Title string `xorm:"varchar(250) not null" json:"title" valid:"required,runelength(1|250)" minLength:"1" maxLength:"250" doc:"The title of the project."`
	// The description of the project.
	Description string `xorm:"longtext null" json:"description" doc:"The description of the project."`
	// The unique project short identifier. Used to build task identifiers.
	Identifier string `xorm:"varchar(10) null" json:"identifier" valid:"runelength(0|10)" minLength:"0" maxLength:"10" doc:"The unique project short identifier."`
	// The hex color of this project
	HexColor string `xorm:"varchar(6) null" json:"hex_color" valid:"runelength(0|7)" maxLength:"7" doc:"The hex color of this project, without the leading #."`

	OwnerID         int64    `xorm:"bigint INDEX not null" json:"-"`
	ParentProjectID int64    `xorm:"bigint INDEX null" json:"parent_project_id" doc:"The id of the parent project. 0 if this is a top-level project."`
	ParentProject   *Project `xorm:"-" json:"-"`

	// The user who created this project.
	Owner *user.User `xorm:"-" json:"owner" valid:"-" readOnly:"true" doc:"The user who owns this project."`

	// Whether a project is archived.
	IsArchived bool `xorm:"not null default false" json:"is_archived" query:"is_archived" doc:"Whether the project is archived."`

	// A timestamp when this project was created.
	Created time.Time `xorm:"created not null" json:"created" readOnly:"true" doc:"A timestamp when this project was created."`
	// A timestamp when this project was last updated.
	Updated time.Time `xorm:"updated not null" json:"updated" readOnly:"true" doc:"A timestamp when this project was last updated."`

	web.CRUDable    `xorm:"-" json:"-"`
	web.Permissions `xorm:"-" json:"-"`
}

// TableName returns a better name for the projects table
func (p *Project) TableName() string {
	return "projects"
}

// projectIdentifierRe matches identifiers like PROJ, PROJ_1, P2.
var projectIdentifierRe = regexp.MustCompile(`^[A-Z][A-Z0-9_]{0,9}$`)

// ValidateProjectIdentifier checks the format of a project short identifier.
func ValidateProjectIdentifier(identifier string) error {
	if !projectIdentifierRe.MatchString(identifier) {
		return ErrInvalidProjectIdentifier{Identifier: identifier}
	}
	return nil
}

// ReadAll gets all projects a user has access to
func (p *Project) ReadAll(s *xorm.Session, a web.Auth, search string, page int, perPage int) (result interface{}, resultCount int, totalItems int64, err error) {
	doer, err := user.GetFromAuth(a)
	if err != nil {
		return nil, 0, 0, err
	}

	prs, resultCount, totalItems, err := getRawProjectsForUser(
		s,
		&projectOptions{
			search:      search,
			page:        page,
			perPage:     perPage,
			getArchived: p.IsArchived,
			user:        doer,
		},
	)
	if err != nil {
		return nil, 0, 0, err
	}

	ownerIDs := make([]int64, len(prs))
	for i, pr := range prs {
		ownerIDs[i] = pr.OwnerID
	}
	owners, err := user.GetUsersByIDs(s, ownerIDs)
	if err != nil {
		return nil, 0, 0, err
	}
	for _, pr := range prs {
		if o, ok := owners[pr.OwnerID]; ok {
			pr.Owner = o
		}
	}

	return prs, resultCount, totalItems, err
}

// ReadOne gets one project by its ID
func (p *Project) ReadOne(s *xorm.Session, a web.Auth) (err error) {
	p.Owner, err = user.GetUserByID(s, p.OwnerID)
	if user.IsErrUserDoesNotExist(err) {
		p.Owner = nil
	} else if err != nil {
		return err
	}

	if !p.IsArchived {
		if err := p.CheckIsArchived(s); err != nil {
			p.IsArchived = true
		}
	}
	return nil
}

// GetProjectSimpleByID gets a project with only the basic items.
func GetProjectSimpleByID(s *xorm.Session, projectID int64) (project *Project, err error) {
	if projectID < 1 {
		return nil, ErrProjectDoesNotExist{ID: projectID}
	}
	project, exists, err := getProjectSimple(s, builder.Eq{"id": projectID})
	if !exists {
		return nil, ErrProjectDoesNotExist{ID: projectID}
	}
	return
}

// GetProjectSimpleByIdentifier gets a project by its textual identifier.
func GetProjectSimpleByIdentifier(s *xorm.Session, identifier string) (project *Project, err error) {
	project, exists, err := getProjectSimple(s, builder.Eq{"identifier": strings.ToUpper(identifier)})
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, ErrProjectDoesNotExist{}
	}
	return
}

func getProjectSimple(s *xorm.Session, cond builder.Cond) (project *Project, exists bool, err error) {
	project = &Project{}
	exists, err = s.Where(cond).OrderBy("id").Get(project)
	return
}

// GetProjectSimpleByTaskID gets a project by a task id
func GetProjectSimpleByTaskID(s *xorm.Session, taskID int64) (l *Project, err error) {
	var project Project
	exists, err := s.
		Select("projects.*").
		Table(Project{}).
		Join("INNER", "tasks", "projects.id = tasks.project_id").
		Where("tasks.id = ?", taskID).
		Get(&project)
	if err != nil {
		return
	}
	if !exists {
		return nil, ErrProjectDoesNotExist{}
	}
	return &project, nil
}

// GetProjectsMapByIDs returns a map of projects from a slice with project ids
func GetProjectsMapByIDs(s *xorm.Session, projectIDs []int64) (projects map[int64]*Project, err error) {
	projects = make(map[int64]*Project, len(projectIDs))
	if len(projectIDs) == 0 {
		return
	}
	err = s.In("id", projectIDs).Find(&projects)
	return
}

func GetProjectsByIDs(s *xorm.Session, projectIDs []int64) (projects []*Project, err error) {
	projects = make([]*Project, 0, len(projectIDs))
	if len(projectIDs) == 0 {
		return
	}
	err = s.In("id", projectIDs).Find(&projects)
	return
}

type projectOptions struct {
	search      string
	user        *user.User
	page        int
	perPage     int
	getArchived bool
}

func getRawProjectsForUser(s *xorm.Session, opts *projectOptions) (projects []*Project, resultCount int, totalItems int64, err error) {
	limit, start := getLimitFromPageIndex(opts.page, opts.perPage)

	projects = []*Project{}
	query := s.Where("owner_id = ?", opts.user.ID).And("is_archived = ?", opts.getArchived)
	if opts.search != "" {
		query = query.And("title LIKE ?", "%"+opts.search+"%")
	}
	totalItems, err = query.Limit(limit, start).FindAndCount(&projects)
	if err != nil {
		return nil, 0, 0, err
	}

	return projects, len(projects), totalItems, nil
}

// CheckIsArchived checks if the project is archived via its parent project.
func (p *Project) CheckIsArchived(s *xorm.Session) (err error) {
	if p.IsArchived {
		return nil
	}

	if p.ParentProjectID == 0 {
		return ErrProjectDoesNotExist{ID: p.ID}
	}

	parent := &Project{ID: p.ParentProjectID}
	exists, err := s.Get(parent)
	if err != nil {
		return err
	}
	if !exists {
		return ErrProjectDoesNotExist{ID: p.ParentProjectID}
	}
	if parent.IsArchived {
		return nil
	}

	return parent.CheckIsArchived(s)
}

// CanCreate checks if the user can create a new project.
func (p *Project) CanCreate(s *xorm.Session, a web.Auth) (bool, error) {
	return true, nil
}

// CanRead checks if the user has read access to a project.
func (p *Project) CanRead(s *xorm.Session, a web.Auth) (bool, int, error) {
	if a == nil {
		return false, 0, nil
	}

	u, err := user.GetFromAuth(a)
	if err != nil {
		return false, 0, err
	}

	if p.OwnerID == u.ID {
		return true, 0, nil
	}

	pu := &ProjectUser{UserID: u.ID, ProjectID: p.ID}
	exists, err := s.Get(pu)
	if err != nil {
		return false, 0, err
	}
	return exists, 0, nil
}

// CanWrite checks if a user has write access to a project.
func (p *Project) CanWrite(s *xorm.Session, a web.Auth) (bool, error) {
	can, _, err := p.CanRead(s, a)
	return can, err
}

// CanUpdate checks if the user can update a project.
func (p *Project) CanUpdate(s *xorm.Session, a web.Auth) (bool, error) {
	return p.CanWrite(s, a)
}

// CanDelete checks if the user can delete a project.
func (p *Project) CanDelete(s *xorm.Session, a web.Auth) (bool, error) {
	can, _, err := p.CanRead(s, a)
	return can, err
}

// Create creates a new project.
func (p *Project) Create(s *xorm.Session, a web.Auth) (err error) {
	doer, err := user.GetFromAuth(a)
	if err != nil {
		return err
	}

	if p.Title == "" {
		return ErrProjectCannotBeEmpty{}
	}

	p.OwnerID = doer.ID
	p.HexColor = utils.NormalizeHex(p.HexColor)
	p.Identifier = strings.ToUpper(p.Identifier)
	if p.Identifier != "" {
		if err := ValidateProjectIdentifier(p.Identifier); err != nil {
			return err
		}
		exists, err := s.Where("identifier = ?", p.Identifier).Exist(&Project{})
		if err != nil {
			return err
		}
		if exists {
			return ErrProjectIdentifierIsNotUnique{Identifier: p.Identifier}
		}
	}

	_, err = s.Insert(p)
	if err != nil {
		return err
	}

	// Make the creator an admin on their project
	pu := &ProjectUser{
		UserID:    doer.ID,
		ProjectID: p.ID,
	}
	_, err = s.Insert(pu)
	if err != nil {
		return err
	}

	events.DispatchOnCommit(s, &ProjectCreatedEvent{
		Project: p,
		Doer:    doer,
	})

	return updateProjectLastUpdated(s, p)
}

// Update updates a project.
func (p *Project) Update(s *xorm.Session, a web.Auth) (err error) {
	oldProject, err := GetProjectSimpleByID(s, p.ID)
	if err != nil {
		return err
	}

	p.HexColor = utils.NormalizeHex(p.HexColor)
	if p.Identifier != "" {
		p.Identifier = strings.ToUpper(p.Identifier)
		if err := ValidateProjectIdentifier(p.Identifier); err != nil {
			return err
		}
		exists, err := s.Where("identifier = ?", p.Identifier).And("id != ?", p.ID).Exist(&Project{})
		if err != nil {
			return err
		}
		if exists {
			return ErrProjectIdentifierIsNotUnique{Identifier: p.Identifier}
		}
	}

	title := oldProject.Title
	if p.Title != "" {
		title = p.Title
	}

	_, err = s.ID(p.ID).Cols("title", "description", "identifier", "hex_color", "parent_project_id", "is_archived").Update(&Project{
		Title:           title,
		Description:     p.Description,
		Identifier:      p.Identifier,
		HexColor:        p.HexColor,
		ParentProjectID: p.ParentProjectID,
		IsArchived:      p.IsArchived,
	})
	if err != nil {
		return err
	}

	doer, err := user.GetFromAuth(a)
	if err != nil {
		return err
	}

	events.DispatchOnCommit(s, &ProjectUpdatedEvent{
		Project: p,
		Doer:    doer,
	})

	return updateProjectLastUpdated(s, &Project{ID: p.ID})
}

// Delete deletes a project.
func (p *Project) Delete(s *xorm.Session, a web.Auth) (err error) {
	doer, err := user.GetFromAuth(a)
	if err != nil {
		return err
	}

	events.DispatchOnCommit(s, &ProjectDeletedEvent{
		Project: p,
		Doer:    doer,
	})

	_, err = s.ID(p.ID).Delete(p)
	return err
}

func updateProjectLastUpdated(s *xorm.Session, project *Project) error {
	_, err := s.ID(project.ID).Cols("updated").Update(project)
	return err
}

func updateProjectLastUpdatedByTaskID(s *xorm.Session, taskID int64) (err error) {
	project, err := GetProjectSimpleByTaskID(s, taskID)
	if err != nil {
		return err
	}
	return updateProjectLastUpdated(s, project)
}

// GenerateProjectIdentifier creates a valid, unique project short identifier from a title.
func GenerateProjectIdentifier(s *xorm.Session, title string, excludeProjectID int64) (string, error) {
	base := strings.ToUpper(title)
	re := regexp.MustCompile(`[^A-Z0-9_]+`)
	base = re.ReplaceAllString(base, "_")
	base = strings.Trim(base, "_")

	if base == "" {
		base = "P"
	} else if base[0] >= '0' && base[0] <= '9' {
		base = "P" + base
	}

	candidate := base
	if len(candidate) > 10 {
		candidate = candidate[:10]
	}
	candidate = strings.TrimRight(candidate, "_")

	for i := 1; i < 1000; i++ {
		exists, err := s.Where("identifier = ?", candidate).And("id != ?", excludeProjectID).Exist(&Project{})
		if err != nil {
			return "", err
		}
		if !exists {
			return candidate, nil
		}

		suffix := fmt.Sprintf("_%d", i)
		prefix := base
		if len(prefix)+len(suffix) > 10 {
			prefix = prefix[:10-len(suffix)]
			prefix = strings.TrimRight(prefix, "_")
		}
		candidate = prefix + suffix
	}

	return "", errors.New("could not generate a unique project identifier")
}
