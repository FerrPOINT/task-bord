<template>
	<Multiselect
		class="control is-expanded"
		:placeholder="$t('project.search')"
		:search-results="foundProjects"
		label="title"
		:select-placeholder="$t('project.searchSelect')"
		:model-value="project"
		@update:modelValue="(val) => val === null ? select(null) : Object.assign(project, val)"
		@select="select"
		@search="findProjects"
	>
		<template #searchResult="{option}">
			<span v-if="projectStore.getAncestors(option).length > 1" class="has-text-grey">
				{{ projectStore.getAncestors(option).filter(p => p.id !== option.id).map(p => getProjectTitle(p)).join(' > ') }} >
			</span>
			{{ getProjectTitle(option) }}
		</template>
	</Multiselect>
</template>

<script lang="ts" setup>
// @ts-nocheck
import {reactive, ref, watch} from 'vue'

import type {IProject} from '@/modelTypes/IProject'

import {useProjectStore} from '@/stores/projects'
import {getProjectTitle} from '@/helpers/getProjectTitle'

import ProjectModel from '@/models/project'
import Multiselect from '@/components/input/Multiselect.vue'

const props = withDefaults(defineProps<{
	modelValue?: IProject
	filter?: (project: IProject) => boolean,
}>(), {
	modelValue: () => new ProjectModel(),
	filter: () => true,
})

const emit = defineEmits<{
	'update:modelValue': [value: IProject | null]
}>()

const project: IProject = reactive(new ProjectModel())

watch(
	() => props.modelValue,
	(newProject) => Object.assign(project, newProject),
	{
		immediate: true,
		deep: true,
	},
)

const projectStore = useProjectStore()

const foundProjects = ref<IProject[]>([])
function findProjects(query: string) {
	if (query === '') {
		select(null)
	}

	const found = projectStore.searchProject(query)
	foundProjects.value = found.filter(props.filter)
}

function select(p: IProject | null) {
	if (p === null) {
		Object.assign(project, new ProjectModel())
		emit('update:modelValue', null)
		return
	}
	Object.assign(project, p)
	emit('update:modelValue', project)
}
</script>
