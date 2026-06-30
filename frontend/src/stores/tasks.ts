// @ts-nocheck
import {computed, ref} from 'vue'
import {acceptHMRUpdate, defineStore} from 'pinia'
import router from '@/router'

import TaskService from '@/services/task'
import TaskAssigneeService from '@/services/taskAssignee'
import LabelTaskService from '@/services/labelTask'

import TaskAssigneeModel from '@/models/taskAssignee'
import LabelTask from '@/models/labelTask'
import TaskModel from '@/models/task'
import LabelModel from '@/models/label'

import type {ILabel} from '@/modelTypes/ILabel'
import type {ITask} from '@/modelTypes/ITask'
import type {IUser} from '@/modelTypes/IUser'
import type {IProject} from '@/modelTypes/IProject'

import {setModuleLoading} from '@/stores/helper'
import {useLabelStore} from '@/stores/labels'
import {useProjectStore} from '@/stores/projects'
import {useKanbanStore} from '@/stores/kanban'
import {useBaseStore} from '@/stores/base'
import ProjectUserService from '@/services/projectUsers'
import {useAuthStore} from '@/stores/auth'

export const useTaskStore = defineStore('task', () => {
	const authStore = useAuthStore()
	const baseStore = useBaseStore()
	const projectStore = useProjectStore()
	const labelStore = useLabelStore()
	const kanbanStore = useKanbanStore()

	const taskDetail = ref<ITask | null>(null)
	const isLoading = ref(false)

	function setLoading(value: boolean) {
		isLoading.value = value
	}

	function setTaskDetail(task: ITask | null) {
		taskDetail.value = task
	}

	async function loadTaskDetail(taskId: ITask['id']) {
		const cancel = setModuleLoading(setLoading)
		const taskService = new TaskService()
		try {
			const task = await taskService.get({id: taskId})
			setTaskDetail(task)
			return task
		} finally {
			cancel()
		}
	}

	async function updateTask(task: Partial<ITask> & {id: ITask['id']}) {
		const taskService = new TaskService()
		const updated = await taskService.update(new TaskModel(task))
		if (taskDetail.value?.id === task.id) {
			setTaskDetail(updated)
		}
		kanbanStore.setTask(updated)
		return updated
	}

	async function createTask(task: Partial<ITask>) {
		const taskService = new TaskService()
		const created = await taskService.create(new TaskModel(task))
		if (created.projectId === baseStore.currentProject?.id) {
			kanbanStore.tasks.push(created)
		}
		return created
	}

	async function deleteTask(taskId: ITask['id']) {
		const taskService = new TaskService()
		await taskService.delete({id: taskId})
		kanbanStore.removeTask(taskId)
		if (taskDetail.value?.id === taskId) {
			setTaskDetail(null)
		}
	}

	async function addAssignee(task: ITask, user: IUser) {
		const assignee = new TaskAssigneeModel({
			taskId: task.id,
			userId: user.id,
		})
		const service = new TaskAssigneeService()
		await service.create(assignee)
		task.assignees.push(user)
		return task
	}

	async function removeAssignee(task: ITask, user: IUser) {
		const service = new TaskAssigneeService()
		await service.delete({
			taskId: task.id,
			userId: user.id,
		})
		task.assignees = task.assignees.filter(a => a.id !== user.id)
		return task
	}

	async function addLabel(task: ITask, label: ILabel) {
		const labelTask = new LabelTask({
			taskId: task.id,
			labelId: label.id,
		})
		const labelTaskService = new LabelTaskService()
		await labelTaskService.create(labelTask)
		task.labels.push(label)
		return task
	}

	async function removeLabel(task: ITask, label: ILabel) {
		const labelTaskService = new LabelTaskService()
		await labelTaskService.delete({
			taskId: task.id,
			labelId: label.id,
		})
		task.labels = task.labels.filter(l => l.id !== label.id)
		return task
	}

	return {
		taskDetail,
		isLoading: readonly(isLoading),
		setTaskDetail,
		loadTaskDetail,
		updateTask,
		createTask,
		deleteTask,
		addAssignee,
		removeAssignee,
		addLabel,
		removeLabel,
	}
})

// support hot reloading
if (import.meta.hot) {
	import.meta.hot.accept(acceptHMRUpdate(useTaskStore, import.meta.hot))
}
