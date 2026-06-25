// @ts-nocheck
import {computed, reactive, toRefs} from 'vue'
import {acceptHMRUpdate, defineStore} from 'pinia'
import {parseURL} from 'ufo'

import {HTTPFactory} from '@/helpers/fetcher'
import {objectToCamelCase} from '@/helpers/case'

import {InvalidApiUrlProvidedError} from '@/helpers/checkAndSetApiUrl'

export interface ConfigState {
	version: string,
	frontendUrl: string,
	motd: string,
	maxFileSize: string,
	maxItemsPerPage: number,
	taskAttachmentsEnabled: boolean,
	legal: {
		imprintUrl: string,
		privacyPolicyUrl: string,
	},
	userDeletionEnabled: boolean,
	taskCommentsEnabled: boolean,
	demoModeEnabled: boolean,
	auth: {
		local: {
			enabled: boolean,
			registrationEnabled: boolean,
		},
	},
	publicTeamsEnabled: boolean,
	allowIconChanges: boolean,
	concurrentWrites: boolean,
}

export const useConfigStore = defineStore('config', () => {
	const state: ConfigState = reactive({
		// These are the api defaults.
		version: '',
		frontendUrl: '',
		motd: '',
		maxFileSize: '20MB',
		maxItemsPerPage: 50,
		taskAttachmentsEnabled: true,
		legal: {
			imprintUrl: '',
			privacyPolicyUrl: '',
		},
		userDeletionEnabled: true,
		taskCommentsEnabled: true,
		demoModeEnabled: false,
		auth: {
			local: {
				enabled: true,
				registrationEnabled: true,
			},
		},
		publicTeamsEnabled: false,
		allowIconChanges: true,
		concurrentWrites: false,
	})

	const apiBase = computed(() => {
		const {host, protocol, pathname} = parseURL(window.API_URL)
		// Strip the /api/v1 suffix (and optional trailing slash) to get the deployment base.
		const basePath = pathname
			.replace(/\/api\/v1\/?$/, '')
			.replace(/\/+$/, '')
		return `${protocol}//${host}${basePath}`
	})

	function setConfig(config: ConfigState) {
		Object.assign(state, config)
	}

	async function update(): Promise<boolean> {
		const HTTP = HTTPFactory()
		const {data: config} = await HTTP.get('info')
		if (typeof config.version === 'undefined') {
			throw new InvalidApiUrlProvidedError()
		}

		setConfig(objectToCamelCase(config) as ConfigState)
		return !!config
	}

	return {
		...toRefs(state),
		apiBase,
		setConfig,
		update,
	}
})

// support hot reloading
if (import.meta.hot) {
	import.meta.hot.accept(acceptHMRUpdate(useConfigStore, import.meta.hot))
}
