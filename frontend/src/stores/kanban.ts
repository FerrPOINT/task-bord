// @ts-nocheck
import {computed, readonly, ref} from 'vue'
import {acceptHMRUpdate, defineStore} from 'pinia'

import TaskCollectionService, {type TaskFilterParams} from '@/services/taskCollection'
import {setModuleLoading} from '@/stores/helper'

import type {ITask} from '@/modelTypes/ITask'
import type {IProject} from '@/modelTypes/IProject'

const TASKS_PER_PAGE = 25

/**
 * MVP kanban store: keeps the task list for the currently selected project.
 */
export const useKanbanStore = defineStore('kanban', () => {
	const tasks = ref<ITask[]>([])
	const projectId = ref<IProject['id']>(0)
	const isLoading = ref(false)
	const page = ref(1)
	const totalPages = ref(1)
	const allLoaded = ref(false)

	const getTaskById = computed(() => {
		return (id: ITask['id']) => tasks.value.find(t => t.id === id) || null
	})

	function setIsLoading(newIsLoading: boolean) {
		isLoading.value = newIsLoading
	}

	function setProjectId(newProjectId: IProject['id']) {
		projectId.value = Number(newProjectId)
	}

	function setTasks(newTasks: ITask[]) {
		tasks.value = newTasks
	}

	function appendTasks(newTasks: ITask[]) {
		tasks.value.push(...newTasks)
	}

	function setTask(task: ITask) {
		const idx = tasks.value.findIndex(t => t.id === task.id)
		if (idx !== -1) {
			tasks.value[idx] = task
		}
	}

	function removeTask(taskId: ITask['id']) {
		const idx = tasks.value.findIndex(t => t.id === taskId)
		if (idx !== -1) {
			tasks.value.splice(idx, 1)
		}
	}

	async function loadTasksForProject(newProjectId: IProject['id'], params?: TaskFilterParams) {
		const cancel = setModuleLoading(setIsLoading)
		setProjectId(newProjectId)
		page.value = 1
		allLoaded.value = false
		setTasks([])

		const service = new TaskCollectionService()
		try {
			const result = await service.getAll({projectId: newProjectId}, {
				...params,
				sort_by: ['id'],
				order_by: ['asc'],
				per_page: TASKS_PER_PAGE,
			}, 1)
			setTasks(result)
			totalPages.value = service.totalPages || 1
			if (page.value >= totalPages.value) {
				allLoaded.value = true
			}
			return result
		} finally {
			cancel()
		}
	}

	async function loadNextTasks(params?: TaskFilterParams) {
		if (isLoading.value || allLoaded.value) {
			return
		}
		const nextPage = page.value + 1
		const cancel = setModuleLoading(setIsLoading)
		const service = new TaskCollectionService()
		try {
			const result = await service.getAll({projectId: projectId.value}, {
				...params,
				sort_by: ['id'],
				order_by: ['asc'],
				per_page: TASKS_PER_PAGE,
			}, nextPage)
			appendTasks(result)
			page.value = nextPage
			if (page.value >= totalPages.value) {
				allLoaded.value = true
			}
			return result
		} finally {
			cancel()
		}
	}

	return {
		tasks,
		projectId,
		isLoading: readonly(isLoading),
		allLoaded,
		getTaskById,
		setProjectId,
		setTasks,
		setTask,
		removeTask,
		loadTasksForProject,
		loadNextTasks,
	}
})

// support hot reloading
if (import.meta.hot) {
	import.meta.hot.accept(acceptHMRUpdate(useKanbanStore, import.meta.hot))
}
