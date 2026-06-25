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
	"encoding/json"

	"github.com/FerrPOINT/task-bord/pkg/db"
	"github.com/FerrPOINT/task-bord/pkg/events"
	"github.com/FerrPOINT/task-bord/pkg/log"
	"github.com/FerrPOINT/task-bord/pkg/notifications"
	"github.com/FerrPOINT/task-bord/pkg/user"

	"github.com/ThreeDotsLabs/watermill/message"
	"xorm.io/builder"
	"xorm.io/xorm"
)

// RegisterListeners registers all event listeners
func RegisterListeners() {
	events.RegisterListener((&TaskCommentCreatedEvent{}).Name(), &SendTaskCommentNotification{})
	events.RegisterListener((&TaskAssigneeCreatedEvent{}).Name(), &SendTaskAssignedNotification{})
	events.RegisterListener((&TaskDeletedEvent{}).Name(), &SendTaskDeletedNotification{})
	events.RegisterListener((&ProjectCreatedEvent{}).Name(), &SendProjectCreatedNotification{})
	events.RegisterListener((&TeamMemberAddedEvent{}).Name(), &SendTeamMemberAddedNotification{})
	events.RegisterListener((&TeamMemberRemovedEvent{}).Name(), &CleanupTaskAssignmentsAfterTeamRemoval{})
	events.RegisterListener((&TaskCommentUpdatedEvent{}).Name(), &HandleTaskCommentEditMentions{})
	events.RegisterListener((&TaskCreatedEvent{}).Name(), &HandleTaskCreateMentions{})
	events.RegisterListener((&TaskUpdatedEvent{}).Name(), &HandleTaskUpdatedMentions{})
	events.RegisterListener((&UserDataExportRequestedEvent{}).Name(), &HandleUserDataExport{})
	events.RegisterListener((&TaskCommentCreatedEvent{}).Name(), &HandleTaskUpdateLastUpdated{})
	events.RegisterListener((&TaskCommentUpdatedEvent{}).Name(), &HandleTaskUpdateLastUpdated{})
	events.RegisterListener((&TaskCommentDeletedEvent{}).Name(), &HandleTaskUpdateLastUpdated{})
	events.RegisterListener((&TaskAssigneeCreatedEvent{}).Name(), &HandleTaskUpdateLastUpdated{})
	events.RegisterListener((&TaskAssigneeDeletedEvent{}).Name(), &HandleTaskUpdateLastUpdated{})
	events.RegisterListener((&TaskAttachmentCreatedEvent{}).Name(), &HandleTaskUpdateLastUpdated{})
	events.RegisterListener((&TaskAttachmentDeletedEvent{}).Name(), &HandleTaskUpdateLastUpdated{})
	events.RegisterListener((&TaskRelationCreatedEvent{}).Name(), &HandleTaskUpdateLastUpdated{})
	events.RegisterListener((&TaskRelationDeletedEvent{}).Name(), &HandleTaskUpdateLastUpdated{})
	events.RegisterListener((&TaskCreatedEvent{}).Name(), &UpdateTaskInSavedFilterViews{})
	events.RegisterListener((&TaskUpdatedEvent{}).Name(), &UpdateTaskInSavedFilterViews{})
	events.RegisterListener((&TaskCommentCreatedEvent{}).Name(), &MarkTaskUnreadOnComment{})
}



//////
// Task Events

func notifyMentionedUsers(sess *xorm.Session, task *Task, text string, n notifications.NotificationWithSubject) (users map[int64]*user.User, err error) {
	users, err = FindMentionedUsersInText(sess, text)
	if err != nil {
		return
	}

	if len(users) == 0 {
		return
	}

	log.Debugf("Processing %d mentioned users for text %d", len(users), n.SubjectID())

	var notified int
	for _, u := range users {
		can, _, err := task.CanRead(sess, u)
		if err != nil {
			return users, err
		}

		if !can {
			continue
		}

		// Don't notify a user if they were already notified
		dbn, err := notifications.GetNotificationsForNameAndUser(sess, u.ID, n.Name(), n.SubjectID())
		if err != nil {
			return users, err
		}

		if len(dbn) > 0 {
			continue
		}

		err = notifications.Notify(u, n, sess)
		if err != nil {
			return users, err
		}
		notified++
	}

	log.Debugf("Notified %d mentioned users for text %d", notified, n.SubjectID())

	return
}

// SendTaskCommentNotification  represents a listener
type SendTaskCommentNotification struct {
}

// Name defines the name for the SendTaskCommentNotification listener
func (s *SendTaskCommentNotification) Name() string {
	return "task.comment.notification.send"
}

// Handle is executed when the event SendTaskCommentNotification listens on is fired
func (s *SendTaskCommentNotification) Handle(msg *message.Message) (err error) {
	event := &TaskCommentCreatedEvent{}
	err = json.Unmarshal(msg.Payload, event)
	if err != nil {
		return err
	}

	sess := db.NewSession()
	defer sess.Close()

	project, err := GetProjectSimpleByID(sess, event.Task.ProjectID)
	if err != nil {
		return err
	}

	n := &TaskCommentNotification{
		Doer:      event.Doer,
		Task:      event.Task,
		Comment:   event.Comment,
		Mentioned: true,
		Project:   project,
	}
	mentionedUsers, err := notifyMentionedUsers(sess, event.Task, event.Comment.Comment, n)
	if err != nil {
		return err
	}

	// Authors of comments quoted via <blockquote data-comment-id="…"> are
	// treated as implicit mentions, sharing the same notification, dedup,
	// permission and subscription logic.
	quotedAuthors, err := findQuotedCommentAuthors(sess, event.Task.ID, event.Doer.ID, event.Comment.Comment)
	if err != nil {
		return err
	}
	for _, u := range quotedAuthors {
		if _, has := mentionedUsers[u.ID]; has {
			continue
		}

		can, _, err := event.Task.CanRead(sess, u)
		if err != nil {
			return err
		}
		if !can {
			continue
		}

		dbn, err := notifications.GetNotificationsForNameAndUser(sess, u.ID, n.Name(), n.SubjectID())
		if err != nil {
			return err
		}
		if len(dbn) > 0 {
			continue
		}

		err = notifications.Notify(u, n, sess)
		if err != nil {
			return err
		}

		if mentionedUsers == nil {
			mentionedUsers = make(map[int64]*user.User)
		}
		mentionedUsers[u.ID] = u
	}

	subscribers, err := GetSubscriptionsForEntity(sess, SubscriptionEntityTask, event.Task.ID)
	if err != nil {
		return err
	}

	log.Debugf("Sending task comment notifications to %d subscribers for task %d", len(subscribers), event.Task.ID)

	for _, subscriber := range subscribers {
		if subscriber.UserID == event.Doer.ID {
			continue
		}

		if _, has := mentionedUsers[subscriber.UserID]; has {
			continue
		}

		n := &TaskCommentNotification{
			Doer:    event.Doer,
			Task:    event.Task,
			Comment: event.Comment,
			Project: project,
		}
		err = notifications.Notify(subscriber.User, n, sess)
		if err != nil {
			return
		}
	}

	return sess.Commit()
}

// HandleTaskCommentEditMentions  represents a listener
type HandleTaskCommentEditMentions struct {
}

// Name defines the name for the HandleTaskCommentEditMentions listener
func (s *HandleTaskCommentEditMentions) Name() string {
	return "handle.task.comment.edit.mentions"
}

// Handle is executed when the event HandleTaskCommentEditMentions listens on is fired
func (s *HandleTaskCommentEditMentions) Handle(msg *message.Message) (err error) {
	event := &TaskCommentUpdatedEvent{}
	err = json.Unmarshal(msg.Payload, event)
	if err != nil {
		return err
	}

	if event.Task == nil || event.Comment == nil {
		return nil
	}

	sess := db.NewSession()
	defer sess.Close()

	project, err := GetProjectSimpleByID(sess, event.Task.ProjectID)
	if err != nil {
		return err
	}

	n := &TaskCommentNotification{
		Doer:      event.Doer,
		Task:      event.Task,
		Comment:   event.Comment,
		Mentioned: true,
		Project:   project,
	}
	_, err = notifyMentionedUsers(sess, event.Task, event.Comment.Comment, n)
	if err != nil {
		return err
	}
	return sess.Commit()
}

// SendTaskAssignedNotification  represents a listener
type SendTaskAssignedNotification struct {
}

// Name defines the name for the SendTaskAssignedNotification listener
func (s *SendTaskAssignedNotification) Name() string {
	return "task.assigned.notification.send"
}

// Handle is executed when the event SendTaskAssignedNotification listens on is fired
func (s *SendTaskAssignedNotification) Handle(msg *message.Message) (err error) {
	event := &TaskAssigneeCreatedEvent{}
	err = json.Unmarshal(msg.Payload, event)
	if err != nil {
		return err
	}

	sess := db.NewSession()
	defer sess.Close()

	subscribers, err := GetSubscriptionsForEntity(sess, SubscriptionEntityTask, event.Task.ID)
	if err != nil {
		return err
	}

	log.Debugf("Sending task assigned notifications to %d subscribers for task %d", len(subscribers), event.Task.ID)

	task, err := GetTaskByIDSimple(sess, event.Task.ID)
	if err != nil {
		return err
	}

	project, err := GetProjectSimpleByID(sess, task.ProjectID)
	if err != nil {
		return err
	}

	notifiedUsers := make(map[int64]bool)

	for _, subscriber := range subscribers {
		if subscriber.UserID == event.Doer.ID {
			continue
		}

		if notifiedUsers[subscriber.UserID] {
			// Users may be subscribed to the task and the project itself, which leads to double notifications
			continue
		}

		n := &TaskAssignedNotification{
			Doer:     event.Doer,
			Task:     &task,
			Assignee: event.Assignee,
			Target:   subscriber.User,
			Project:  project,
		}
		err = notifications.Notify(subscriber.User, n, sess)
		if err != nil {
			return
		}

		notifiedUsers[subscriber.UserID] = true
	}

	return sess.Commit()
}

// SendTaskDeletedNotification  represents a listener
type SendTaskDeletedNotification struct {
}

// Name defines the name for the SendTaskDeletedNotification listener
func (s *SendTaskDeletedNotification) Name() string {
	return "task.deleted.notification.send"
}

// Handle is executed when the event SendTaskDeletedNotification listens on is fired
func (s *SendTaskDeletedNotification) Handle(msg *message.Message) (err error) {
	event := &TaskDeletedEvent{}
	err = json.Unmarshal(msg.Payload, event)
	if err != nil {
		return err
	}

	sess := db.NewSession()
	defer sess.Close()

	var subscribers []*SubscriptionWithUser
	subscribers, err = GetSubscriptionsForEntity(sess, SubscriptionEntityTask, event.Task.ID)
	// If the task does not exist and no one has explicitly subscribed to it, we won't find any subscriptions for it.
	// Hence, we need to check for subscriptions to the parent project manually.
	if err != nil && (IsErrTaskDoesNotExist(err) || IsErrProjectDoesNotExist(err)) {
		subscribers, err = GetSubscriptionsForEntity(sess, SubscriptionEntityProject, event.Task.ProjectID)
	}
	if err != nil {
		return err
	}

	log.Debugf("Sending task deleted notifications to %d subscribers for task %d", len(subscribers), event.Task.ID)

	for _, subscriber := range subscribers {
		if subscriber.UserID == event.Doer.ID {
			continue
		}

		n := &TaskDeletedNotification{
			Doer: event.Doer,
			Task: event.Task,
		}
		err = notifications.Notify(subscriber.User, n, sess)
		if err != nil {
			return
		}
	}

	return sess.Commit()
}

// HandleTaskCreateMentions  represents a listener
type HandleTaskCreateMentions struct {
}

// Name defines the name for the HandleTaskCreateMentions listener
func (s *HandleTaskCreateMentions) Name() string {
	return "task.created.mentions"
}

// Handle is executed when the event HandleTaskCreateMentions listens on is fired
func (s *HandleTaskCreateMentions) Handle(msg *message.Message) (err error) {
	event := &TaskCreatedEvent{}
	err = json.Unmarshal(msg.Payload, event)
	if err != nil {
		return err
	}

	if event.Task == nil {
		return nil
	}

	sess := db.NewSession()
	defer sess.Close()

	project, err := GetProjectSimpleByID(sess, event.Task.ProjectID)
	if err != nil {
		return err
	}

	n := &UserMentionedInTaskNotification{
		Task:    event.Task,
		Doer:    event.Doer,
		IsNew:   true,
		Project: project,
	}
	_, err = notifyMentionedUsers(sess, event.Task, event.Task.Description, n)
	if err != nil {
		return err
	}
	return sess.Commit()
}

// HandleTaskUpdatedMentions  represents a listener
type HandleTaskUpdatedMentions struct {
}

// Name defines the name for the HandleTaskUpdatedMentions listener
func (s *HandleTaskUpdatedMentions) Name() string {
	return "task.updated.mentions"
}

// Handle is executed when the event HandleTaskUpdatedMentions listens on is fired
func (s *HandleTaskUpdatedMentions) Handle(msg *message.Message) (err error) {
	event := &TaskUpdatedEvent{}
	err = json.Unmarshal(msg.Payload, event)
	if err != nil {
		return err
	}

	if event.Task == nil {
		return nil
	}

	sess := db.NewSession()
	defer sess.Close()

	project, err := GetProjectSimpleByID(sess, event.Task.ProjectID)
	if err != nil {
		return err
	}

	n := &UserMentionedInTaskNotification{
		Task:    event.Task,
		Doer:    event.Doer,
		IsNew:   false,
		Project: project,
	}

	_, err = notifyMentionedUsers(sess, event.Task, event.Task.Description, n)
	if err != nil {
		return err
	}
	return sess.Commit()
}

// HandleTaskUpdateLastUpdated  represents a listener
type HandleTaskUpdateLastUpdated struct {
}

// Name defines the name for the HandleTaskUpdateLastUpdated listener
func (s *HandleTaskUpdateLastUpdated) Name() string {
	return "handle.task.update.last.updated"
}

// Handle is executed when the event HandleTaskUpdateLastUpdated listens on is fired
func (s *HandleTaskUpdateLastUpdated) Handle(msg *message.Message) (err error) {
	// Using a map here allows us to plug this listener to all kinds of task events
	event := map[string]interface{}{}
	err = json.Unmarshal(msg.Payload, &event)
	if err != nil {
		return err
	}

	task, is := event["task"].(map[string]interface{})
	if !is {
		log.Errorf("Event payload does not contain task")
		return
	}

	taskID, is := task["id"]
	if !is {
		log.Errorf("Event payload does not contain a valid task ID")
		return
	}

	var taskIDInt int64
	switch v := taskID.(type) {
	case int64:
		taskIDInt = v
	case int:
		taskIDInt = int64(v)
	case int32:
		taskIDInt = int64(v)
	case float64:
		taskIDInt = int64(v)
	case float32:
		taskIDInt = int64(v)
	default:
		log.Errorf("Event payload does not contain a valid task ID")
		return
	}

	sess := db.NewSession()
	defer sess.Close()

	err = updateTaskLastUpdated(sess, &Task{ID: taskIDInt})
	if err != nil {
		return err
	}

	return sess.Commit()
}

// UpdateTaskInSavedFilterViews  represents a listener
type UpdateTaskInSavedFilterViews struct {
}

// Name defines the name for the UpdateTaskInSavedFilterViews listener
func (l *UpdateTaskInSavedFilterViews) Name() string {
	return "task.set.saved.filter.views"
}

// Handle is executed when the event UpdateTaskInSavedFilterViews listens on is fired
func (l *UpdateTaskInSavedFilterViews) Handle(msg *message.Message) (err error) {
	event := &TaskCreatedEvent{}
	err = json.Unmarshal(msg.Payload, event)
	if err != nil {
		return err
	}

	if event.Task == nil {
		return nil
	}

	// This operation is potentially very resource-heavy, because we don't know if a task is included
	// in a filter until we evaluate that filter. We need to evaluate each filter individually - since
	// there can be many filters, this can take a while to execute.
	// For this reason, we do this in an asynchronous event listener.

	s := db.NewSession()
	defer s.Close()

	// Get all saved filters with a manual kanban view
	kanbanFilterViews := []*ProjectView{}
	err = s.Where("project_id < 0 and view_kind = ? and bucket_configuration_mode = ?", ProjectViewKindKanban, BucketConfigurationModeManual).
		Find(&kanbanFilterViews)
	if err != nil {
		return err
	}

	filterIDs := []int64{}
	for _, view := range kanbanFilterViews {
		filterIDs = append(filterIDs, GetSavedFilterIDFromProjectID(view.ProjectID))
	}

	filters := map[int64]*SavedFilter{}
	err = s.In("id", filterIDs).Find(&filters)
	if err != nil {
		return err
	}

	var fallbackTimezone string
	if event.Doer != nil {
		u, userErr := user.GetUserByID(s, event.Doer.GetID())
		if userErr == nil {
			fallbackTimezone = u.Timezone
		}
		// When a link share triggered this event, the user id will be 0, and thus this fails.
		// Similarly, when the doer has been deleted, the user will not exist.
		// Only passing the value along when the user was retrieved successfully ensures the whole handler
		// does not fail because of that.
		// When the fallback is empty, it will be handled later anyhow.
	}

	taskBuckets := []*TaskBucket{}
	taskPositions := []*TaskPosition{}

	viewIDToCleanUp := []int64{}

	for _, view := range kanbanFilterViews {
		filter, exists := filters[GetSavedFilterIDFromProjectID(view.ProjectID)]
		if !exists {
			log.Debugf("Did not find filter for view %d", view.ID)
			continue
		}

		taskBucket, taskPosition, err := addTaskToFilter(s, filter, view, fallbackTimezone, event.Task)
		if err != nil {
			if IsErrInvalidFilterExpression(err) ||
				IsErrInvalidTaskFilterValue(err) ||
				IsErrInvalidTaskFilterConcatinator(err) ||
				IsErrInvalidTaskFilterComparator(err) ||
				IsErrInvalidTaskField(err) {
				log.Debugf("Invalid filter expression for view %d, expression: %v", view.ID, view.Filter)
				continue
			}

			return err
		}

		if taskBucket != nil && taskPosition != nil {
			taskBuckets = append(taskBuckets, taskBucket)
			taskPositions = append(taskPositions, taskPosition)
			viewIDToCleanUp = append(viewIDToCleanUp, view.ID)
		}
	}

	if len(taskBuckets) > 0 || len(taskPositions) > 0 {
		_, err = s.And(
			builder.Eq{"task_id": event.Task.ID},
			builder.In("project_view_id", viewIDToCleanUp),
		).
			Delete(&TaskBucket{})
		if err != nil {
			return
		}
		_, err = s.And(
			builder.Eq{"task_id": event.Task.ID},
			builder.In("project_view_id", viewIDToCleanUp),
		).
			Delete(&TaskPosition{})
		if err != nil {
			return
		}
		_, err = s.Insert(taskBuckets)
		if err != nil {
			return
		}
		_, err = s.Insert(taskPositions)
		if err != nil {
			return
		}
	}

	return s.Commit()
}

///////
// Project Event Listeners

// SendProjectCreatedNotification  represents a listener
type SendProjectCreatedNotification struct {
}

// Name defines the name for the SendProjectCreatedNotification listener
func (s *SendProjectCreatedNotification) Name() string {
	return "send.project.created.notification"
}

// Handle is executed when the event SendProjectCreatedNotification listens on is fired
func (s *SendProjectCreatedNotification) Handle(msg *message.Message) (err error) {
	event := &ProjectCreatedEvent{}
	err = json.Unmarshal(msg.Payload, event)
	if err != nil {
		return err
	}

	sess := db.NewSession()
	defer sess.Close()

	subscribers, err := GetSubscriptionsForEntity(sess, SubscriptionEntityProject, event.Project.ID)
	if err != nil {
		return err
	}

	log.Debugf("Sending project created notifications to %d subscribers for project %d", len(subscribers), event.Project.ID)

	for _, subscriber := range subscribers {
		if subscriber.UserID == event.Doer.ID {
			continue
		}

		n := &ProjectCreatedNotification{
			Doer:    event.Doer,
			Project: event.Project,
		}
		err = notifications.Notify(subscriber.User, n, sess)
		if err != nil {
			return
		}
	}

	return sess.Commit()
}

// Team Events

// CleanupTaskAssignmentsAfterTeamRemoval represents a listener
type CleanupTaskAssignmentsAfterTeamRemoval struct{}

// Name defines the name of the listener
func (l *CleanupTaskAssignmentsAfterTeamRemoval) Name() string {
	return "task.assignees.cleanup.team_removal"
}

// Handle cleans up task assignments and subscriptions for members removed from teams
func (l *CleanupTaskAssignmentsAfterTeamRemoval) Handle(msg *message.Message) (err error) {
	event := &TeamMemberRemovedEvent{}
	err = json.Unmarshal(msg.Payload, event)
	if err != nil {
		return err
	}

	s := db.NewSession()
	defer s.Close()

	if event == nil || event.Team == nil || event.Member == nil {
		return nil
	}

	err = cleanupTaskMembersAfterTeamRemoval(s, event.Team.ID, event.Member.ID)
	if err != nil {
		_ = s.Rollback()
		return err
	}

	return s.Commit()
}

// SendTeamMemberAddedNotification  represents a listener
type SendTeamMemberAddedNotification struct {
}

// Name defines the name for the SendTeamMemberAddedNotification listener
func (s *SendTeamMemberAddedNotification) Name() string {
	return "team.member.added.notification"
}

// Handle is executed when the event SendTeamMemberAddedNotification listens on is fired
func (s *SendTeamMemberAddedNotification) Handle(msg *message.Message) (err error) {
	event := &TeamMemberAddedEvent{}
	err = json.Unmarshal(msg.Payload, event)
	if err != nil {
		return err
	}

	// Don't notify the user themselves
	if event.Doer.ID == event.Member.ID {
		return nil
	}

	return notifications.Notify(event.Member, &TeamMemberAddedNotification{
		Member: event.Member,
		Doer:   event.Doer,
		Team:   event.Team,
	})
}

// HandleUserDataExport  represents a listener
type HandleUserDataExport struct {
}

// Name defines the name for the HandleUserDataExport listener
func (s *HandleUserDataExport) Name() string {
	return "handle.user.data.export"
}

// Handle is executed when the event HandleUserDataExport listens on is fired
func (s *HandleUserDataExport) Handle(msg *message.Message) (err error) {
	event := &UserDataExportRequestedEvent{}
	err = json.Unmarshal(msg.Payload, event)
	if err != nil {
		return err
	}

	log.Debugf("Starting to export user data for user %d...", event.User.ID)

	sess := db.NewSession()
	defer sess.Close()

	err = ExportUserData(sess, event.User)
	if err != nil {
		_ = sess.Rollback()
		return
	}

	log.Debugf("Done exporting user data for user %d...", event.User.ID)

	return sess.Commit()
}

type MarkTaskUnreadOnComment struct {
}

func (s *MarkTaskUnreadOnComment) Name() string {
	return "task.comment.mark.unread"
}

func (s *MarkTaskUnreadOnComment) Handle(msg *message.Message) (err error) {
	event := &TaskCommentCreatedEvent{}
	err = json.Unmarshal(msg.Payload, event)
	if err != nil {
		return err
	}

	sess := db.NewSession()
	defer sess.Close()

	project, err := GetProjectSimpleByID(sess, event.Task.ProjectID)
	if err != nil {
		_ = sess.Rollback()
		return err
	}

	users, err := ListUsersFromProject(sess, project, event.Doer, "")
	if err != nil {
		_ = sess.Rollback()
		return err
	}

	// Get existing unread statuses for this task
	existingUnreadStatuses := []*TaskUnreadStatus{}
	err = sess.
		Where("task_id = ?", event.Task.ID).
		Find(&existingUnreadStatuses)
	if err != nil {
		_ = sess.Rollback()
		return err
	}

	// Create a set of existing user IDs for quick lookup
	existingUserIDs := make(map[int64]bool)
	for _, status := range existingUnreadStatuses {
		existingUserIDs[status.UserID] = true
	}

	// Build list of new unread statuses
	unreadStatuses := []*TaskUnreadStatus{}
	for _, u := range users {
		// Skip the comment author and users who already have unread status
		if u.ID == event.Doer.ID || existingUserIDs[u.ID] {
			continue
		}
		unreadStatuses = append(unreadStatuses, &TaskUnreadStatus{
			TaskID: event.Task.ID,
			UserID: u.ID,
		})
	}

	// Bulk insert new unread statuses
	if len(unreadStatuses) > 0 {
		_, err = sess.Insert(&unreadStatuses)
		if err != nil {
			_ = sess.Rollback()
			return err
		}
	}

	return sess.Commit()
}