<template>
	<header
		:class="{ 'has-background': background, 'menu-active': menuActive }"
		aria-label="main navigation"
		class="navbar d-print-none"
	>
		<RouterLink
			:to="{ name: 'home' }"
			class="logo-link"
			:aria-label="$t('navigation.home')"
		>
			<Logo
				width="164"
				height="48"
			/>
		</RouterLink>

		<MenuButton class="menu-button" />

		<div
			v-if="currentProject?.id"
			class="project-title-wrapper"
		>
			<span class="project-title">
				{{ currentProject.title === '' ? $t('misc.loading') : getProjectTitle(currentProject) }}
			</span>

			<BaseButton
				v-if="!isEditorContentEmpty(currentProject.description)"
				:to="{ name: 'project.info', params: { projectId: currentProject.id } }"
				class="project-title-button"
			>
				<span class="is-sr-only">{{ $t('project.description') }}</span>
				<Icon icon="circle-info" />
			</BaseButton>
		</div>

		<div
			v-else-if="pageTitle"
			class="project-title-wrapper"
		>
			<span class="project-title">{{ pageTitle }}</span>
		</div>

		<div class="navbar-end">
			<OpenQuickActions />
			<BaseButton @click="authStore.logout()">
				{{ $t('user.auth.logout') }}
			</BaseButton>
		</div>
	</header>
</template>

<script setup lang="ts">
// @ts-nocheck


import { computed } from 'vue'
import { useRoute } from 'vue-router'
import { useI18n } from 'vue-i18n'

import Logo from '@/components/home/Logo.vue'
import BaseButton from '@/components/base/BaseButton.vue'
import MenuButton from '@/components/home/MenuButton.vue'
import OpenQuickActions from '@/components/misc/OpenQuickActions.vue'

import { getProjectTitle } from '@/helpers/getProjectTitle'
import { isEditorContentEmpty } from '@/helpers/editorContentEmpty'

import { useBaseStore } from '@/stores/base'
import { useAuthStore } from '@/stores/auth'
import type { IProject } from '@/modelTypes/IProject'

const baseStore = useBaseStore()
const currentProject = computed<IProject | null>(() => {
	const project = baseStore.currentProject
	return project ? { ...project } as IProject : null
})
const background = computed(() => baseStore.background)
const menuActive = computed(() => baseStore.menuActive)

const route = useRoute()
const { t } = useI18n()
const pageTitle = computed(() => {
	const title = route.meta.title as string | undefined
	return title ? t(title) : ''
})

const authStore = useAuthStore()
</script>

<style lang="scss" scoped>
$navbar-height: 64px;

.navbar {
	--navbar-button-min-width: 40px;
	--navbar-gap-width: 1rem;
	--navbar-icon-size: 1.25rem;

	position: fixed;
	inset-block-start: 0;
	inset-inline-start: 0;
	inset-inline-end: 0;
	z-index: 30;

	display: flex;
	justify-content: space-between;
	gap: var(--navbar-gap-width);
	min-block-size: $navbar-height;

	background: var(--site-background);

	@media screen and (min-width: $tablet) {
		padding-inline-start: 2rem;
		align-items: stretch;
	}

	&.menu-active {
		@media screen and (max-width: $tablet) {
			z-index: 0;
		}
	}
}

.logo-link {
	display: none;

	@media screen and (min-width: $tablet) {
		align-self: stretch;
		display: flex;
		align-items: center;
		margin-inline-end: .5rem;
	}
}

.menu-button {
	margin-inline-end: auto;
	align-self: stretch;
	flex: 0 0 auto;

	@media screen and (max-width: $tablet) {
		margin-inline-start: 1rem;
	}
}

.project-title-wrapper {
	margin-inline: auto;
	display: flex;
	align-items: center;

	min-inline-size: 0;

	@media screen and (min-width: $tablet) {
		padding-inline: var(--navbar-gap-width);
	}
}

.project-title {
	font-size: 1rem;
	text-overflow: ellipsis;
	overflow: hidden;
	white-space: nowrap;

	@media screen and (min-width: $tablet) {
		font-size: 1.75rem;
	}
}

.project-title-button {
	align-self: stretch;
	min-inline-size: var(--navbar-button-min-width);
	display: flex;
	place-items: center;
	justify-content: center;
	font-size: var(--navbar-icon-size);
	color: var(--grey-400);
}

.navbar-end {
	flex: 0 0 auto;
	display: flex;
	align-items: stretch;
	gap: .5rem;

	>* {
		min-inline-size: var(--navbar-button-min-width);
	}
}
</style>
