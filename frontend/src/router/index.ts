import {createRouter, createWebHistory} from 'vue-router'
import type {RouteLocation} from 'vue-router'
import {saveLastVisited} from '@/helpers/saveLastVisited'

import {getProjectViewId} from '@/helpers/projectView'
import {parseDateOrString} from '@/helpers/time/parseDateOrString'
import {getNextWeekDate} from '@/helpers/time/getNextWeekDate'

import {useAuthStore} from '@/stores/auth'
import {useBaseStore} from '@/stores/base'

import Login from '@/views/user/Login.vue'
import Register from '@/views/user/Register.vue'
import UpcomingTasks from '@/views/tasks/ShowTasks.vue'

import NotFoundComponent from '@/views/404.vue'

const AUTH_ROUTE_NAMES = new Set([
	'user.login',
	'user.register',
])

const REDIRECT_HASH_PREFIX = '#redirect='

const router = createRouter({
	history: createWebHistory(import.meta.env.BASE_URL),
	scrollBehavior(to, from, savedPosition) {
		if (savedPosition) {
			return savedPosition
		}

		if (to.hash) {
			return {el: to.hash}
		}

		return {
			'inset-inline-start': 0,
			'inset-block-start': 0,
		}
	},
	routes: [
		{
			path: '/',
			name: 'home',
			component: () => import('@/views/Home.vue'),
		},
		{
			path: '/tasks/:identifierOrId',
			name: 'task.detail',
			component: () => import('@/views/tasks/TaskDetailView.vue'),
			props: route => ({identifierOrId: route.params.identifierOrId as string}),
		},
		{
			path: '/login',
			name: 'user.login',
			component: Login,
			meta: {
				title: 'user.auth.login',
			},
		},
		{
			path: '/register',
			name: 'user.register',
			component: Register,
			meta: {
				title: 'user.auth.createAccount',
			},
		},
		{
			path: '/tasks/by/upcoming',
			name: 'tasks.range',
			component: UpcomingTasks,
			props: route => ({
				dateFrom: parseDateOrString(route.query.from as string, new Date()),
				dateTo: parseDateOrString(route.query.to as string, getNextWeekDate()),
				showNulls: route.query.showNulls === 'true',
				showOverdue: route.query.showOverdue === 'true',
			}),
		},
		{
			path: '/lists:pathMatch(.*)*',
			name: 'lists',
			redirect(to) {
				return {
					path: to.path.replace('/lists', '/projects'),
					query: to.query,
					hash: to.hash,
				}
			},
		},
		{
			path: '/projects',
			name: 'projects.index',
			component: () => import('@/views/project/ListProjects.vue'),
		},
		{
			path: '/projects/new',
			name: 'project.create',
			component: () => import('@/views/project/NewProject.vue'),
			meta: {
				showAsModal: true,
			},
		},
		{
			path: '/projects/:parentProjectId/new',
			name: 'project.createFromParent',
			component: () => import('@/views/project/NewProject.vue'),
			props: route => ({parentProjectId: Number(route.params.parentProjectId as string)}),
			meta: {
				showAsModal: true,
			},
		},
		{
			path: '/projects/:projectId/settings/edit',
			name: 'project.settings.edit',
			component: () => import('@/views/project/settings/ProjectSettingsEdit.vue'),
			props: route => ({projectId: Number(route.params.projectId as string)}),
			meta: {
				showAsModal: true,
			},
		},
		{
			path: '/projects/:projectId/settings/delete',
			name: 'project.settings.delete',
			component: () => import('@/views/project/settings/ProjectSettingsDelete.vue'),
			meta: {
				showAsModal: true,
			},
		},
		{
			path: '/projects/:projectId/settings/archive',
			name: 'project.settings.archive',
			component: () => import('@/views/project/settings/ProjectSettingsArchive.vue'),
			meta: {
				showAsModal: true,
			},
		},
		{
			path: '/projects/:projectId/info',
			name: 'project.info',
			component: () => import('@/views/project/ProjectInfo.vue'),
			meta: {
				showAsModal: true,
			},
			props: route => ({projectId: Number(route.params.projectId as string)}),
		},
		{
			path: '/projects/:projectId',
			name: 'project.index',
			redirect(to) {
				const viewId = getProjectViewId(Number(to.params.projectId as string))
				return {
					name: 'project.view',
					params: {
						projectId: parseInt(to.params.projectId as string),
						viewId: viewId ?? 0,
					},
				}
			},
		},
		{
			path: '/projects/:projectId/:viewId',
			name: 'project.view',
			component: () => import('@/views/project/ProjectView.vue'),
			props: route => ({
				projectId: parseInt(route.params.projectId as string),
				viewId: route.params.viewId ? parseInt(route.params.viewId as string) : undefined,
			}),
		},
		{
			path: '/labels',
			name: 'labels.index',
			component: () => import('@/views/labels/ListLabels.vue'),
		},
		{
			path: '/labels/new',
			name: 'labels.create',
			component: () => import('@/views/labels/NewLabel.vue'),
			meta: {
				showAsModal: true,
			},
		},
		{
			path: '/about',
			name: 'about',
			component: () => import('@/views/About.vue'),
		},
		{
			path: '/:pathMatch(.*)*',
			name: 'not-found',
			component: NotFoundComponent,
		},
		{
			path: '/:pathMatch(.*)',
			name: 'bad-not-found',
			component: NotFoundComponent,
		},
	],
})

export async function getAuthForRoute(to: RouteLocation, authStore: {authUser: unknown, authLinkShare: unknown}) {
	const redirectDest = to.name === 'user.login' && to.hash.startsWith(REDIRECT_HASH_PREFIX)
		? to.hash.slice(REDIRECT_HASH_PREFIX.length)
		: ''

	if (authStore.authUser || authStore.authLinkShare) {
		if (redirectDest) {
			return redirectDest
		}
		return
	}

	const isValidUserAppRoute = !AUTH_ROUTE_NAMES.has(to.name as string)

	if (isValidUserAppRoute) {
		saveLastVisited(to.name as string, to.params, to.query)
	}

	if (isValidUserAppRoute) {
		return {name: 'user.login'}
	}
}

router.beforeEach(async (to) => {
	const authStore = useAuthStore()

	await authStore.checkAuth()

	const newRoute = await getAuthForRoute(to, authStore)
	if (newRoute) {
		if (typeof newRoute === 'string') {
			return newRoute
		}
		return {
			hash: to.hash,
			...newRoute,
		}
	}

	if (to.hash.startsWith(REDIRECT_HASH_PREFIX)) {
		return
	}

	if (!to.fullPath.endsWith(to.hash)) {
		return to.fullPath + to.hash
	}
})

export default router
