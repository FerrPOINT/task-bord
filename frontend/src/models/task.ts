// @ts-nocheck
import AbstractModel from '@/models/abstractModel'
import type {ITask} from '@/modelTypes/ITask'

export default class TaskModel extends AbstractModel implements ITask {
	id = 0
	title = ''
	description = ''
	done = false
	doneAt = null
	dueDate = null
	projectId = 0
	index = 0
	identifier = ''
	created = null
	updated = null
	createdBy = null
	createdById = 0
	assignees = []
	labels = []
	comments = []
	commentCount = 0

	constructor(data: Partial<ITask> = {}) {
		super()
		this.assignData(data)
	}
}
