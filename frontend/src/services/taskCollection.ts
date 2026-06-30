// @ts-nocheck
import AbstractService from '@/services/abstractService'
import TaskModel from '@/models/task'

import type {ITask} from '@/modelTypes/ITask'

export interface TaskFilterParams {
	sort_by?: string[],
	order_by?: string[],
	filter?: string,
	filter_include_nulls?: boolean,
	filter_timezone?: string,
	s?: string,
	per_page?: number,
}

export default class TaskCollectionService extends AbstractService<ITask> {
	constructor() {
		super({
			getAll: '/projects/{projectId}/tasks',
		})
	}

	modelFactory(data) {
		return new TaskModel(data)
	}
}
