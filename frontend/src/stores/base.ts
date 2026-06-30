// @ts-nocheck
import {ref, computed, readonly} from 'vue'
import {useI18n} from 'vue-i18n'
import {defineStore, acceptHMRUpdate} from 'pinia'

import ProjectService from '@/services/project'
import {checkAndSetApiUrl, ERROR_NO_API_URL, InvalidApiUrlProvidedError, NoApiUrlProvidedError} from '@/helpers/checkAndSetApiUrl'

import {useMenuActive} from '@/composables/useMenuActive'

import {useAuthStore} from '@/stores/auth'
import router from '@/router'
import type {IProject} from '@/modelTypes/IProject'

export const useBaseStore = defineStore('base', () => {
	const authStore = useAuthStore()
	
	const {t} = useI18n()

	const ready = ref(false)
	const error = ref('')
	const loading = computed(() => !ready.value && error.value === '')

	// This is used to highlight the current project in menu for all project related views
	const currentProject = ref<IProject | null>(null)
	const hasTasks = ref(false)
	const keyboardShortcutsActive = ref(false)
	const quickActionsActive = ref(false)
	const logoVisible = ref(true)
	const updateAvailable = ref(false)

	function setCurrentProject(newCurrentProject: IProject | null) {
		currentProject.value = newCurrentProject
	}

	function setHasTasks(newHasTasks: boolean) {
		hasTasks.value = newHasTasks
	}

	function setKeyboardShortcutsActive(value: boolean) {
		keyboardShortcutsActive.value = value
	}

	function setQuickActionsActive(value: boolean) {
		quickActionsActive.value = value
	}

	function setLogoVisible(visible: boolean) {
		logoVisible.value = visible
	}
	
	function setUpdateAvailable(value: boolean) {
		updateAvailable.value = value
	}

	async function handleSetCurrentProject(
		{project}: {project: IProject | null},
	) {
		if (project === null || typeof project === 'undefined') {
			setCurrentProject(null)
			return
		}

		setCurrentProject(project)
	}

	async function handleSetCurrentProjectIfNotSet(project: IProject) {
		if (currentProject.value?.id !== project.id) {
			await handleSetCurrentProject({project})
		}
	}

	async function hydrateConfig() {
		try {
			await checkAndSetApiUrl(window.API_URL)
			await authStore.checkAuth()
		} catch (e: unknown) {
			if (e instanceof NoApiUrlProvidedError) {
				error.value = ERROR_NO_API_URL
				return
			}
			if (e instanceof InvalidApiUrlProvidedError) {
				error.value = t('apiConfig.error')
				return
			}
			error.value = String(e instanceof Error ? e.message : e)
		}
	}

	// Exposed so router guards can await config/auth hydration on direct
	// navigation without deadlocking on router.isReady().
	const appReady = hydrateConfig()

	async function loadApp() {
		// Re-hydrates (used when the user selects a new API URL from Ready.vue).
		await hydrateConfig()
		await router.isReady()
		ready.value = true
	}

	// Initial load: wait on the in-flight hydration, then mark ready once
	// the router has settled.
	appReady.then(async () => {
		await router.isReady()
		ready.value = true
	})

	return {
		error: readonly(error),
		loading: readonly(loading),
		ready: readonly(ready),
		loadApp,
		appReady,

		currentProject: readonly(currentProject),
		hasTasks: readonly(hasTasks),
		keyboardShortcutsActive: readonly(keyboardShortcutsActive),
		quickActionsActive: readonly(quickActionsActive),
		logoVisible: readonly(logoVisible),
		updateAvailable: readonly(updateAvailable),

		setCurrentProject,
		setHasTasks,
		setKeyboardShortcutsActive,
		setQuickActionsActive,
		setLogoVisible,
		setUpdateAvailable,

		handleSetCurrentProject,
		handleSetCurrentProjectIfNotSet,

		...useMenuActive(),
	}
})

// support hot reloading
if (import.meta.hot) {
	import.meta.hot.accept(acceptHMRUpdate(useBaseStore, import.meta.hot))
}
