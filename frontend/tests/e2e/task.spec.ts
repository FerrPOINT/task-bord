import {test, expect} from '@playwright/test'
import {getAuthToken, authHeaders} from './helpers/auth'

test('create and move a task via API', async ({request}) => {
	const {token} = await getAuthToken(request)
	const h = authHeaders(token)

	// Create a project
	const projectRes = await request.post('/api/v1/projects', {
		headers: h,
		data: {title: `E2E Task Project ${Date.now()}`},
	})
	expect(projectRes.ok()).toBeTruthy()
	const project = await projectRes.json()
	expect(project.id).toBeGreaterThan(0)

	// Create a kanban view with manual buckets
	const viewRes = await request.post(`/api/v1/projects/${project.id}/views`, {
		headers: h,
		data: {
			title: 'Board',
			view_kind: 'kanban',
			bucket_configuration_mode: 'manual',
		},
	})
	expect(viewRes.ok()).toBeTruthy()
	const view = await viewRes.json()
	expect(view.id).toBeGreaterThan(0)

	// List buckets
	const bucketsRes = await request.get(`/api/v1/projects/${project.id}/views/${view.id}/buckets/tasks`, {
		headers: h,
	})
	expect(bucketsRes.ok()).toBeTruthy()
	const bucketsBody = await bucketsRes.json()
	const buckets = bucketsBody.items || []
	expect(buckets.length).toBeGreaterThanOrEqual(2)

	// Create a task
	const taskRes = await request.post(`/api/v1/projects/${project.id}/tasks`, {
		headers: h,
		data: {title: `E2E Task ${Date.now()}`, description: 'Created by E2E test'},
	})
	expect(taskRes.ok()).toBeTruthy()
	const task = await taskRes.json()
	expect(task.id).toBeGreaterThan(0)

	// Move task to the second bucket
	const targetBucket = buckets[1]
	const moveRes = await request.put(`/api/v1/projects/${project.id}/views/${view.id}/buckets/${targetBucket.id}/tasks`, {
		headers: h,
		data: {task_id: task.id},
	})
	expect(moveRes.ok()).toBeTruthy()
	const moved = await moveRes.json()
	expect(moved.bucket_id).toBe(targetBucket.id)
})
