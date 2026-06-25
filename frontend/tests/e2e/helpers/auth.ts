import type {APIRequestContext} from '@playwright/test'
import fs from 'node:fs'
import path from 'node:path'
import {fileURLToPath} from 'node:url'

const __dirname = path.dirname(fileURLToPath(import.meta.url))
const TOKEN_PATH = path.join(__dirname, '../.auth/token.json')

export interface AuthToken {
	token: string
	username: string
}

export async function getAuthToken(request: APIRequestContext): Promise<AuthToken> {
	if (fs.existsSync(TOKEN_PATH)) {
		return JSON.parse(fs.readFileSync(TOKEN_PATH, 'utf-8'))
	}

	const username = 'e2euser'
	const password = 'e2ePassword123!'
	const email = 'e2e@example.com'

	// Try to register; 400/409 both mean the user already exists in the container DB.
	const registerRes = await request.post('/api/v1/register', {
		data: {username, password, email, language: 'en'},
	})
	if (!registerRes.ok() && ![400, 409].includes(registerRes.status())) {
		throw new Error(`Registration failed: ${registerRes.status()} ${await registerRes.text()}`)
	}

	const loginRes = await request.post('/api/v1/login', {
		data: {username, password, long_token: true},
	})
	if (!loginRes.ok()) {
		throw new Error(`Login failed: ${loginRes.status()} ${await loginRes.text()}`)
	}

	const {token} = await loginRes.json()
	const auth: AuthToken = {token, username}
	fs.mkdirSync(path.dirname(TOKEN_PATH), {recursive: true})
	fs.writeFileSync(TOKEN_PATH, JSON.stringify(auth))
	return auth
}

export function authHeaders(token: string) {
	return {
		Authorization: `Bearer ${token}`,
	}
}
