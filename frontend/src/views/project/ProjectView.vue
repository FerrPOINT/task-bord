<script setup lang="ts">
// @ts-nocheck
import {computed, ref, watch} from 'vue'
import {useRoute} from 'vue-router'

import {useBaseStore} from '@/stores/base'
import {useProjectStore} from '@/stores/projects'
import {useAuthStore} from '@/stores/auth'
import {useKanbanStore} from '@/stores/kanban'

import ProjectService from '@/services/project'

const props = defineProps<{
	projectId: number,
}>()

const baseStore = useBaseStore()
const projectStore = useProjectStore()
const authStore = useAuthStore()
const kanbanStore = useKanbanStore()
const route = useRoute()

const currentProject = computed(() => projectStore.projects[props.projectId])
const projectService = new ProjectService()
const isLoadingProject = computed(() => projectService.loading)
const loadedProjectId = ref(0)

watch(
	() => props.projectId,
	async (projectIdToLoad, oldProjectIdToLoad) => {
		if (projectIdToLoad !== oldProjectIdToLoad) {
			loadedProjectId.value = 0
		}
		try {
			const loadedProject = await projectService.get({id: projectIdToLoad})
			projectStore.setProject(loadedProject)
			baseStore.handleSetCurrentProject({project: loadedProject})
			await kanbanStore.loadTasksForProject(projectIdToLoad)
		} finally {
			loadedProjectId.value = projectIdToLoad
		}
	},
	{immediate: true},
)

watch(
	() => authStore.authenticated,
	(authenticated) => {
		if (authenticated) {
			// history saving removed in MVP
		}
	},
	{immediate: true},
)
</script>

<template>
	<div class="project-view">
		<div class="loading" v-if="isLoadingProject">
			Loading project...
		</div>
		<div v-else-if="currentProject" class="project-content">
			<h1 class="title">{{ currentProject.title }}</h1>
			<div class="task-list">
				<div v-if="kanbanStore.tasks.length === 0" class="empty">
					No tasks yet.
				</div>
				<div
					v-for="task in kanbanStore.tasks"
					:key="task.id"
					class="task-item"
				>
					<span :class="{'task-done': task.done}">{{ task.title }}</span>
				</div>
			</div>
		</div>
	</div>
</template>

<style scoped>
.project-view {
	padding: 1rem;
}
.title {
	font-size: 1.5rem;
	margin-bottom: 1rem;
}
.task-list {
	display: flex;
	flex-direction: column;
	gap: 0.5rem;
}
.task-item {
	padding: 0.5rem;
	border: 1px solid #ddd;
	border-radius: 4px;
}
.task-done {
	text-decoration: line-through;
	opacity: 0.6;
}
.empty {
	opacity: 0.7;
}
</style>
