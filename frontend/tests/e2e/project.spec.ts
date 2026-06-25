import {test, expect} from '@playwright/test'
import {getAuthToken, authHeaders} from './helpers/auth'

test('create a project via API', async ({request}) => {
	const {token} = await getAuthToken(request)

	const title = `E2E Project ${Date.now()}`
	const res = await request.post('/api/v1/projects', {
		headers: authHeaders(token),
		data: {title},
	})
	expect(res.ok()).toBeTruthy()
	const body = await res.json()
	expect(body.title).toBe(title)
	expect(body.id).toBeGreaterThan(0)
})
