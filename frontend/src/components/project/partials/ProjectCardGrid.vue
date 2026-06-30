<template>
	<ul class="project-grid">
		<li
			v-for="(item, index) in filteredProjects"
			:key="`project_${item.id}_${index}`"
			class="project-grid-item"
		>
			<RouterLink :to="{name: 'project.index', params: {projectId: item.id}}">
				<div class="project-card" :style="{backgroundColor: item.hexColor}">
					<h3 class="project-title">{{ item.title }}</h3>
				</div>
			</RouterLink>
		</li>
	</ul>
</template>

<script lang="ts" setup>
// @ts-nocheck
import {computed} from 'vue'
import type {IProject} from '@/modelTypes/IProject'

const props = withDefaults(defineProps<{
	projects: IProject[],
	showArchived?: boolean,
}>(), {
	showArchived: false,
})

const filteredProjects = computed(() => {
	return props.showArchived
		? props.projects
		: props.projects.filter(l => !l.isArchived)
})
</script>

<style lang="scss" scoped>
.project-grid {
	--project-grid-item-height: 150px;
	--project-grid-gap: 1rem;
	margin: 0;
	list-style-type: none;
	display: grid;
	grid-template-columns: repeat(var(--project-grid-columns, 1), 1fr);
	grid-auto-rows: var(--project-grid-item-height);
	gap: var(--project-grid-gap);
}

@media screen and (min-width: 768px) {
	.project-grid {
		--project-grid-columns: 3;
	}
}

@media screen and (min-width: 1408px) {
	.project-grid {
		--project-grid-columns: 5;
	}
}

.project-card {
	width: 100%;
	height: 100%;
	border-radius: 8px;
	display: flex;
	align-items: center;
	justify-content: center;
	padding: 1rem;
	color: white;
}

.project-title {
	font-size: 1.25rem;
	font-weight: 600;
	text-align: center;
	word-break: break-word;
}
</style>
