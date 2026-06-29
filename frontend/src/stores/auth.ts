import {ref, computed, readonly} from 'vue'
import {acceptHMRUpdate, defineStore} from 'pinia'

import {AuthenticatedHTTPFactory, HTTPFactory} from '@/helpers/fetcher'
import {i18n} from '@/i18n'
import {objectToSnakeCase} from '@/helpers/case'
import UserModel, {getDisplayName} from '@/models/user'
import {getToken, refreshToken, removeToken, saveToken} from '@/helpers/auth'
import router from '@/router'
import {setModuleLoading} from '@/stores/helper'

export const useAuthStore = defineStore('auth', () => {
	const authenticated = ref(false)

	const info = ref<IUser | null>(null)

	const currentSessionId = ref<string | null>(null)
	const lastUserInfoRefresh = ref<Date | null>(null)
	const isLoading = ref(false)

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

	function setUser(newUser: IUser | null) {
		info.value = newUser
	}

	function setAuthenticated(newAuthenticated: boolean) {
		authenticated.value = newAuthenticated
	}

	function updateLastUserRefresh() {
		lastUserInfoRefresh.value = new Date()
	}

	async function login(credentials: {username: string, password: string}) {
		const HTTP = HTTPFactory()
		setIsLoading(true)

		removeToken()

		try {
			const response = await HTTP.post('login', objectToSnakeCase(credentials))
			saveToken(response.data.token, true)
			await checkAuth()
		} finally {
			setIsLoading(false)
		}
	}

	async function register(credentials: {username: string, password: string}, language: string | null = null) {
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
		} catch (e: any) {
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

	async function checkAuth() {
		const now = new Date()
		const oneMinuteAgo = new Date(new Date().setMinutes(now.getMinutes() - 1))
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
				const ts = Math.round((new Date()).getTime() / 1000)
				isAuthenticated = jwtUser.exp >= ts
				currentSessionId.value = payload.sid ?? null
				if (isAuthenticated) {
					if (
						info.value === null ||
						info.value.id !== jwtUser.id
					) {
						setUser(jwtUser)
					} else {
						info.value.exp = jwtUser.exp
					}
				} else if (jwtUser.type === AUTH_TYPES.USER) {
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
								setUser(freshUser)
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

			setUser(newUser)
			updateLastUserRefresh()

			return newUser
		} catch (e: any) {
			if ((e?.response?.status >= 400 && e?.response?.status < 500) ||
				e?.response?.data?.message === 'missing, malformed, expired or otherwise invalid token provided') {
				await logout()
				return
			}

			console.error('Error refreshing user info:', e)
			throw new Error('Error while refreshing user info', {cause: e})
		}
	}

	async function logout() {
		try {
			const HTTP = AuthenticatedHTTPFactory()
			await HTTP.post('user/logout')
		} catch (_e) {
			// Ignore
		}

		removeToken()
		setUser(null)
		setAuthenticated(false)
		router.push({name: 'user.login'})
	}

	function getBrowserLanguage(): string {
		return navigator.language?.split('-')[0] ?? 'en'
	}

	return {
		authenticated: readonly(authenticated),
		info: readonly(info),
		currentSessionId: readonly(currentSessionId),
		lastUserInfoRefresh: readonly(lastUserInfoRefresh),
		authUser,
		userDisplayName,
		isLoading: readonly(isLoading),
		setIsLoading,
		setUser,
		setAuthenticated,
		updateLastUserRefresh,
		login,
		register,
		checkAuth,
		refreshUserInfo,
		logout,
	}
})

if (import.meta.hot) {
	import.meta.hot.accept(acceptHMRUpdate(useAuthStore, import.meta.hot))
}
