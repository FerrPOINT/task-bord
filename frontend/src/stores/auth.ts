// @ts-nocheck
import {computed, readonly, ref} from 'vue'
import {acceptHMRUpdate, defineStore} from 'pinia'

import {AuthenticatedHTTPFactory, HTTPFactory} from '@/helpers/fetcher'
import {DEFAULT_LANGUAGE, i18n, setLanguage} from '@/i18n'
import {objectToSnakeCase} from '@/helpers/case'
import UserModel, {getDisplayName, fetchAvatarBlobUrl, invalidateAvatarCache} from '@/models/user'
import AvatarService from '@/services/avatar'
import UserSettingsService from '@/services/userSettings'
import {getToken, refreshToken, removeToken, saveToken} from '@/helpers/auth'
import {useWebSocket} from '@/composables/useWebSocket'
import router from '@/router'
import {setModuleLoading} from '@/stores/helper'
import {success, error} from '@/message'
import {AUTH_TYPES, type IUser} from '@/modelTypes/IUser'
import type {IUserSettings} from '@/modelTypes/IUserSettings'
import {useConfigStore} from '@/stores/config'
import UserSettingsModel from '@/models/userSettings'
import {MILLISECONDS_A_SECOND} from '@/constants/date'
import {PrefixMode} from '@/modules/quickAddMagic'
import {DATE_DISPLAY} from '@/constants/dateDisplay'
import {TIME_FORMAT} from '@/constants/timeFormat'
import {RELATION_KIND} from '@/types/IRelationKind'



export const useAuthStore = defineStore('auth', () => {
	const configStore = useConfigStore()
	
	const authenticated = ref(false)
	
	const info = ref<IUser | null>(null)
	const avatarUrl = ref('')
	const settings = ref<IUserSettings>(new UserSettingsModel())
	
	const currentSessionId = ref<string | null>(null)
	const lastUserInfoRefresh = ref<Date | null>(null)
	const isLoading = ref(false)
	const isLoadingGeneralSettings = ref(false)

	const authUser = computed(() => {
		return authenticated.value && (
			info.value &&
			info.value.type === AUTH_TYPES.USER
		)
	})

	const userDisplayName = computed(() => info.value ? getDisplayName(info.value) : undefined)

	function setIsLoading(newIsLoading: boolean) {
		isLoading.value = newIsLoading 
	}

	function setIsLoadingGeneralSettings(isLoading: boolean) {
		isLoadingGeneralSettings.value = isLoading 
	}

	function setUser(newUser: IUser | null, saveSettings = true) {
		info.value = newUser
		if (newUser !== null) {
			reloadAvatar()

			if (saveSettings && newUser.settings) {
				loadSettings(newUser.settings)
			}
		}
	}

	function setUserSettings(newSettings: IUserSettings) {
		loadSettings(newSettings)
		info.value = new UserModel({
			...info.value !== null ? info.value : {},
			name: newSettings.name,
		})
	}
	
	function loadSettings(newSettings: IUserSettings) {
		settings.value = new UserSettingsModel({
			...newSettings,
			frontendSettings: {
				// Need to set default settings here in case the user does not have any saved in the api already
				playSoundWhenDone: true,
				quickAddMagicMode: PrefixMode.Default,
				colorSchema: 'auto',
				allowIconChanges: true,
				dateDisplay: DATE_DISPLAY.RELATIVE,
				timeFormat: TIME_FORMAT.HOURS_24,
				defaultTaskRelationType: RELATION_KIND.RELATED,
				backgroundBrightness: 100,
				showLastViewed: true,
				sidebarWidth: null,
				commentSortOrder: 'asc',
				desktopQuickEntryShortcut: 'CmdOrCtrl+Shift+A',
				...newSettings.frontendSettings,
			},
		})

		// Sync the quick entry shortcut to the desktop app when settings are loaded
		window.taskBoardDesktop?.updateQuickEntryShortcut(
			settings.value.frontendSettings.desktopQuickEntryShortcut || '',
		)
	}

	function setAuthenticated(newAuthenticated: boolean) {
		authenticated.value = newAuthenticated
	}

	async function reloadAvatar() {
		if (!info.value || !info.value.username) {
			return
		}
		invalidateAvatarCache(info.value)
		avatarUrl.value = await fetchAvatarBlobUrl(info.value, 40)
	}

	function updateLastUserRefresh() {
		lastUserInfoRefresh.value = new Date()
	}

	// Logs a user in with a set of credentials.
	async function login(credentials) {
		const HTTP = HTTPFactory()
		setIsLoading(true)

		// Delete an eventually preexisting old token
		removeToken()

		try {
			const response = await HTTP.post('login', objectToSnakeCase(credentials))
			// Save the token to local storage for later use
			saveToken(response.data.token, true)

			// Tell others the user is authenticated
			await checkAuth()
		} finally {
			setIsLoading(false)
		}
	}

	/**
	 * Registers a new user and logs them in.
	 * Not sure if this is the right place to put the logic in, maybe a separate js component would be better suited. 
	 */
	async function register(credentials, language: string|null = null) {
		const HTTP = HTTPFactory()
		setIsLoading(true)
		
		if (!language) {
			language = i18n.global.locale.value ?? getBrowserLanguage()
		}
		
		try {
			await HTTP.post('register', {
				...credentials,
				language,
			})
			return login(credentials)
		} catch (e) {
			if (e.response?.data?.code === 2002 && e.response?.data?.invalid_fields[0]?.startsWith('language:')) {
				return register(credentials, 'en')
			}
			
			if (e.response?.data?.message) {
				throw e.response.data
			}

			throw e
		} finally {
			setIsLoading(false)
		}
	}

	/**
	 * Populates user information from jwt token saved in local storage in store
	 */
	async function checkAuth() {
		const now = new Date()
		const oneMinuteAgo = new Date(new Date().setMinutes(now.getMinutes() - 1))
		// This function can be called from multiple places at the same time and shortly after one another.
		// To prevent hitting the api too frequently or race conditions, we check at most once per minute.
		if (
			lastUserInfoRefresh.value !== null &&
			lastUserInfoRefresh.value > oneMinuteAgo
		) {
			return
		}

		const jwt = getToken()
		let isAuthenticated = false
		if (jwt) {
			try {
				const base64 = jwt
					.split('.')[1]
					.replace(/-/g, '+')
					.replace(/_/g, '/')
				const payload = JSON.parse(atob(base64))
				const jwtUser = new UserModel(payload)
				const ts = Math.round((new Date()).getTime() / MILLISECONDS_A_SECOND)
				isAuthenticated = jwtUser.exp >= ts
				currentSessionId.value = payload.sid ?? null
				if (isAuthenticated) {
					if (
						info.value === null ||
						info.value.id !== jwtUser.id
					) {
						setUser(jwtUser, false)
					} else {
						// Always keep exp in sync so token renewal checks stay accurate
						info.value.exp = jwtUser.exp
					}
				} else if (jwtUser.type === AUTH_TYPES.USER) {
					// JWT expired but this is a user session — attempt a cookie-based
					// refresh before giving up. This lets users who reopen the app
					// after the short JWT TTL seamlessly resume their session.
					try {
						await refreshToken(true)
						const freshJwt = getToken()
						if (freshJwt) {
							const b64 = freshJwt.split('.')[1].replace(/-/g, '+').replace(/_/g, '/')
							const p = JSON.parse(atob(b64))
							const freshUser = new UserModel(p)
							isAuthenticated = freshUser.exp >= ts
							currentSessionId.value = p.sid ?? null
							if (info.value === null || info.value.id !== freshUser.id) {
								setUser(freshUser, false)
							} else {
								info.value.exp = freshUser.exp
							}
						}
					} catch {
						// Refresh failed — stay unauthenticated
					}
				}
			} catch (_) {
				logout()
			}

			if (isAuthenticated) {
				const user = await refreshUserInfo()
				if (!user) {
					// refreshUserInfo() did not return a user — either the
					// token vanished or a 4xx triggered logout(). Bail out
					// so the stale local `isAuthenticated` doesn't override
					// the auth state that logout() already set.
					return
				}
			}
		}

		setAuthenticated(isAuthenticated)
		if (!isAuthenticated) {
			setUser(null)
			router.push({name: 'user.login'})
		}
		
		return Promise.resolve(authenticated)
	}

	async function refreshUserInfo() {
		const jwt = getToken()
		if (!jwt) {
			return
		}

		const HTTP = AuthenticatedHTTPFactory()
		try {
			const response = await HTTP.get('user')
			const newUser = new UserModel({
				...response.data,
				...(info.value?.exp && {exp: info.value?.exp}),
			})

			if (newUser.settings.language) {
				// Always keep Russian as the UI language regardless of the user's stored preference.
				await setLanguage(DEFAULT_LANGUAGE)
			}

			setUser(newUser)
			updateLastUserRefresh()

			return newUser
		} catch (e) {
			if((e?.response?.status >= 400 && e?.response?.status < 500) ||
				e?.response?.data?.message === 'missing, malformed, expired or otherwise invalid token provided') {
				await logout()
				return
			}
			
			const cause = {e}
			
			if (typeof e?.response?.data?.message !== 'undefined') {
				cause.message = e.response.data.message
			}
			
			console.error('Error refreshing user info:', e)
			
			throw new Error('Error while refreshing user info:', {cause})
		}
	}

	/**
	 * Try to verify the email
	 */
	async function verifyEmail(): Promise<boolean> {
		const emailVerifyToken = localStorage.getItem('emailConfirmToken')
		if (emailVerifyToken) {
			const stopLoading = setModuleLoading(setIsLoading)
			try {
				await HTTPFactory().post('user/confirm', {token: emailVerifyToken})
				return true
			} catch(e) {
				throw new Error(e.response.data.message)
			} finally {
				localStorage.removeItem('emailConfirmToken')
				stopLoading()
			}
		}
		return false
	}

	async function saveUserSettings({
		settings,
		showMessage = true,
	}: {
		settings: IUserSettings,
		showMessage: boolean,
	}) {
		const userSettingsService = new UserSettingsService()
		const cancel = setModuleLoading(setIsLoadingGeneralSettings)
		try {
			const oldName = info.value?.name
			let settingsUpdate = {...settings}
			if (configStore.demoModeEnabled) {
				settingsUpdate = {
					...settingsUpdate,
					language: null,
				}
			}
			const updateSettingsPromise = userSettingsService.update(settingsUpdate)
			setUserSettings(settingsUpdate)
			await setLanguage(DEFAULT_LANGUAGE)
			await updateSettingsPromise
			if (oldName !== undefined && oldName !== settingsUpdate.name) {
				const {avatarProvider} = await (new AvatarService()).get({})
				if (avatarProvider === 'initials') {
					await reloadAvatar()
				}
			}
			if (showMessage) {
				success({message: i18n.global.t('user.settings.general.savedSuccess')})
			}
		} catch (e) {
			error(e)
		} finally {
			cancel()
		}
	}

	/**
	 * Renews the api token and saves it to local storage
	 */
	async function renewToken() {
		if (!authenticated.value) {
			return
		}

		try {
			// User sessions renew via the refresh-token cookie.
			await refreshToken(true)
			await checkAuth()
		} catch (e) {
			// Only logout if the JWT has actually expired and we can't refresh.
			// If the JWT is still valid, the proactive refresh failure is harmless
			// — the 401 interceptor will handle it when the token really expires.
			const nowInSeconds = Date.now() / MILLISECONDS_A_SECOND
			const isExpired = !info.value?.exp || info.value.exp < nowInSeconds
			if (isExpired && (e?.cause?.request?.status || e?.cause?.response?.status)) {
				await logout()
			}
		}
	}

	async function logout() {
		const {disconnect} = useWebSocket()
		disconnect()

		// Revoke the server session so the refresh token can't be reused.
		// Best-effort: if the network call fails, still clean up locally.
		try {
			const HTTP = AuthenticatedHTTPFactory()
			await HTTP.post('user/logout')
		} catch (_e) {
			// Ignore — session will expire naturally
		}

		removeToken()
	}

	return {
		// state
		authenticated: readonly(authenticated),

		info: readonly(info),
		avatarUrl: readonly(avatarUrl),
		settings: readonly(settings),

		currentSessionId: readonly(currentSessionId),
		lastUserInfoRefresh: readonly(lastUserInfoRefresh),

		authUser,
		userDisplayName,

		isLoading: readonly(isLoading),
		setIsLoading,

		isLoadingGeneralSettings: readonly(isLoadingGeneralSettings),
		setIsLoadingGeneralSettings,

		setUser,
		setUserSettings,
		setAuthenticated,

		reloadAvatar,
		updateLastUserRefresh,

		login,
		register,
		checkAuth,
		refreshUserInfo,
		verifyEmail,
		saveUserSettings,
		renewToken,
		logout,
	}
})

// support hot reloading
if (import.meta.hot) {
	import.meta.hot.accept(acceptHMRUpdate(useAuthStore, import.meta.hot))
}
