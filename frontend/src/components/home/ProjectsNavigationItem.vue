<template>
	<li class="list-menu loader-container is-loading-small" :class="{'is-loading': isLoading}">
		<div class="navigation-item">
			<BaseButton
				v-if="canCollapse && childProjects?.length > 0"
				class="collapse-project-button"
				@click="childProjectsOpen = !childProjectsOpen"
			>
				<Icon icon="chevron-down" :class="{ 'project-is-collapsed': !childProjectsOpen }" />
			</BaseButton>
			<BaseButton
				:to="{ name: 'project.index', params: { projectId: project.id} }"
				class="list-menu-link"
				:class="{'router-link-exact-active': currentProject?.id === project.id}"
			>
				<span
					v-if="!canCollapse || childProjects?.length === 0"
					class="collapse-project-button-placeholder"
				/>
				<div class="color-bubble-wrapper">
					<ColorBubble
						v-if="project.hexColor !== ''"
						:color="project.hexColor"
						:aria-label="$t('project.color')"
					/>
				</div>
				<span class="project-menu-title">{{ getProjectTitle(project) }}</span>
			</BaseButton>
			<BaseButton
				v-if="project.id > 0"
				class="favorite"
				:class="{'is-favorite': project.isFavorite}"
				@click="projectStore.toggleProjectFavorite(project)"
			>
				<span class="is-sr-only">{{ project.isFavorite ? $t('project.unfavorite') : $t('project.favorite') }}</span>
				<Icon :icon="project.isFavorite ? 'star' : ['far', 'star']" />
			</BaseButton>
		</div>
		<ProjectsNavigation
			v-if="childProjectsOpen && canCollapse"
			:model-value="childProjects"
			:can-edit-order="true"
			:can-collapse="canCollapse"
		/>
	</li>
</template>

<script setup lang="ts">
// @ts-nocheck
import {computed, ref} from 'vue'
import {useProjectStore} from '@/stores/projects'
import {useBaseStore} from '@/stores/base'
import {useStorage} from '@vueuse/core'

import type {IProject} from '@/modelTypes/IProject'

import BaseButton from '@/components/base/BaseButton.vue'
import {getProjectTitle} from '@/helpers/getProjectTitle'
import ColorBubble from '@/components/misc/ColorBubble.vue'
import ProjectsNavigation from '@/components/home/ProjectsNavigation.vue'

const props = defineProps<{
	project: IProject,
	isLoading?: boolean,
	canCollapse?: boolean,
	canEditOrder?: boolean,
}>()

const projectStore = useProjectStore()
const baseStore = useBaseStore()
const currentProject = computed(() => baseStore.currentProject)

const childProjectsOpenState = useStorage<{ [key: number]: boolean }>('navigation-child-projects-open', {})
const childProjectsOpen = computed({
	get() {
		return childProjectsOpenState.value[props.project.id] ?? true
	},
	set(open) {
		childProjectsOpenState.value[props.project.id] = open
	},
})

const childProjects = computed(() => {
	return projectStore.getChildProjects(props.project.id)
		.filter(p => !p.isArchived)
})
</script>

<style lang="scss" scoped>
.list-menu {
	transition: background-color $transition;
}

.project-is-collapsed {
	transform: rotate(-90deg);
}

.favorite {
	transition: opacity $transition, color $transition;
	opacity: 0;

	&:hover,
	&.is-favorite {
		opacity: 1;
		color: var(--warning);
	}
}

.list-menu:hover > div > .favorite {
	opacity: 1;
}

.project-menu-title {
	overflow: hidden;
	text-overflow: ellipsis;
}

.color-bubble-wrapper {
	position: relative;
	inline-size: 1rem;
	block-size: 1rem;
	display: flex;
	align-items: center;
	justify-content: flex-start;
	margin-inline-end: .25rem;
	flex-shrink: 0;
}

.list-menu-link {
	display: flex;
	align-items: center;
	flex-grow: 1;
	padding: 0.5rem;
	gap: 0.25rem;
}

.collapse-project-button-placeholder {
	inline-size: 1rem;
	block-size: 1rem;
	display: inline-block;
}
</style>
