<template>
	<div class="content-auth">
		<BaseButton
			v-show="menuActive"
			:aria-label="$t('navigation.closeSidebar')"
			class="menu-hide-button d-print-none"
			@click="baseStore.setMenuActive(false)"
		>
			<Icon icon="times" />
		</BaseButton>
		<div class="app-container">
			<Navigation class="d-print-none" />
			<main
				id="main-content"
				class="app-content"
				:class="[
					{ 'is-menu-enabled': menuActive },
					$route.name,
				]"
				:style="{'--sidebar-width': sidebarWidth}"
			>
				<BaseButton
					v-show="menuActive"
					:aria-label="$t('navigation.closeSidebar')"
					class="mobile-overlay d-print-none"
					@click="baseStore.setMenuActive(false)"
				/>

				<RouterView
					v-slot="{ Component }"
					:route="routeWithModal"
				>
					<component :is="Component" />
				</RouterView>

				<Modal
					:enabled="typeof currentModal !== 'undefined'"
					variant="scrolling"
					class="task-detail-view-modal"
					:aria-label="$t('task.detail.title')"
					@close="closeModal()"
				>
					<component
						:is="currentModal"
						@close="closeModal()"
					/>
				</Modal>
			</main>
		</div>
	</div>
</template>

<script lang="ts" setup>
// @ts-nocheck
import {watch, computed} from 'vue'
import {useRoute} from 'vue-router'

import Navigation from '@/components/home/Navigation.vue'
import BaseButton from '@/components/base/BaseButton.vue'

import {useBaseStore} from '@/stores/base'
import {useLabelStore} from '@/stores/labels'
import {useProjectStore} from '@/stores/projects'

import {useRouteWithModal} from '@/composables/useRouteWithModal'
import {useSidebarResize} from '@/composables/useSidebarResize'

const {sidebarWidth} = useSidebarResize()
const {routeWithModal, currentModal, closeModal} = useRouteWithModal()

const baseStore = useBaseStore()
const menuActive = computed(() => baseStore.menuActive)

const route = useRoute()

watch(() => route.name as string, (routeName) => {
	if (
		routeName &&
		(
			[
				'home',
				'tasks.range',
				'labels.index',
				'projects.index',
			].includes(routeName) ||
			routeName.startsWith('user.settings')
		)
	) {
		baseStore.handleSetCurrentProject({project: null})
	}
})

const labelStore = useLabelStore()
labelStore.loadAllLabels()

const projectStore = useProjectStore()
projectStore.loadAllProjects()
</script>

<style lang="scss" scoped>
.menu-hide-button {
	position: fixed;
	inset-block-start: 0.5rem;
	inset-inline-end: 0.5rem;
	z-index: 31;
	inline-size: 3rem;
	block-size: 3rem;
	display: flex;
	justify-content: center;
	align-items: center;
	font-size: 2rem;
	color: var(--grey-400);
	line-height: 1;
	transition: all $transition;

	@media screen and (min-width: $tablet) {
		display: none;
	}

	&:hover,
	&:focus {
		color: var(--grey-600);
	}
}

.app-container {
	min-block-size: calc(100vh - 65px);

	@media screen and (max-width: $tablet) {
		padding-block-start: $navbar-height;
	}
}

.app-content {
	--sidebar-width: #{$navbar-width};

	display: flow-root;
	z-index: 10;
	position: relative;
	padding: 1.5rem 0.5rem 0;
	transition: margin-inline-start $transition-duration;

	@media screen and (max-width: $tablet) {
		margin-inline-start: 0;
		margin-inline-end: 0;
		min-block-size: calc(100vh - 4rem);
	}

	@media screen and (min-width: $tablet) {
		padding: $navbar-height + 1.5rem 1.5rem 0 1.5rem;
	}

	&.is-menu-enabled {
		@media screen and (min-width: $tablet) {
			margin-inline-start: var(--sidebar-width);
		}
	}
}

.mobile-overlay {
	display: none;
	position: fixed;
	inset-block-start: 0;
	inset-block-end: 0;
	inset-inline-start: 0;
	inset-inline-end: 0;
	block-size: 100vh;
	inline-size: 100vw;
	background: hsla(var(--grey-100-hsl), 0.8);
	z-index: 5;
	opacity: 0;
	transition: all $transition;

	@media screen and (max-width: $tablet) {
		display: block;
		opacity: 1;
	}
}

.content-auth {
	position: relative;
	z-index: 1;
}
</style>
