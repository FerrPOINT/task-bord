// @ts-nocheck
import AbstractService from './abstractService'
import TaskModel from '@/models/task'
import type {ITask} from '@/modelTypes/ITask'
import AttachmentService from './attachment'
import LabelService from './label'

import {colorFromHex} from '@/helpers/color/colorFromHex'
import {objectToSnakeCase} from '@/helpers/case'
import {AuthenticatedHTTPFactory} from '@/helpers/fetcher'

const parseDate = date => {
	if (date) {
		return new Date(date).toISOString()
	}
	return null
}

export default class TaskService extends AbstractService<ITask> {
	constructor() {
		super({
			create: '/projects/{projectId}/tasks',
			getAll: '/tasks',
			get: '/tasks/{id}',
			update: '/tasks/{id}',
			delete: '/tasks/{id}',
		})
	}

	modelFactory(data) {
		return new TaskModel(data)
	}

	beforeUpdate(model) {
		return this.processModel(model)
	}

	beforeCreate(model) {
		return this.processModel(model)
	}

	autoTransformBeforePost(): boolean {
		return false
	}

	processModel(updatedModel) {
		const model = {...updatedModel}
		model.title = model.title?.trim()
		model.projectId = Number(model.projectId)
		model.dueDate = parseDate(model.dueDate)
		model.doneAt = parseDate(model.doneAt)
		model.created = model.created ? new Date(model.created).toISOString() : null
		model.updated = model.updated ? new Date(model.updated).toISOString() : null
		model.hexColor = colorFromHex(model.hexColor)

		if (model.labels?.length > 0) {
			const labelService = new LabelService()
			model.labels = model.labels.map(l => labelService.processModel(l))
		}

		const transformed = objectToSnakeCase(model)
		return transformed as ITask
	}

	async markTaskAsRead(taskId: ITask['id']): Promise<void> {
		const cancel = this.setLoading()
		try {
			await AuthenticatedHTTPFactory().post(`/tasks/${taskId}/read`, {} as ITask)
		} finally {
			cancel()
		}
	}

	async getByIdentifier(identifier: string, {expand = [] as string[]} = {}): Promise<ITask> {
		const cancel = this.setLoading()
		try {
			const response = await AuthenticatedHTTPFactory().get(`${window.location.origin}/api/v1/tasks/by-identifier/${encodeURIComponent(identifier)}`, {
				params: expand.length > 0 ? {expand} : undefined,
			})
			return this.modelFactory(response.data)
		} finally {
			cancel()
		}
	}
}
