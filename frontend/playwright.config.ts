import {defineConfig, devices} from '@playwright/test'
import {execSync} from 'child_process'

// Find system chromium - for UI mode, set PLAYWRIGHT_CHROMIUM_EXECUTABLE_PATH env var
const getChromiumPath = () => {
	// Check if env var is already set (for UI mode)
	if (process.env.PLAYWRIGHT_CHROMIUM_EXECUTABLE_PATH) {
		return process.env.PLAYWRIGHT_CHROMIUM_EXECUTABLE_PATH
	}
	for (const bin of ['chromium', 'chromium-browser', 'google-chrome']) {
		try {
			return execSync(`which ${bin}`, {encoding: 'utf-8'}).trim()
		} catch {
			// try next
		}
	}
	return undefined
}

const baseURL = process.env.BASE_URL || 'http://127.0.0.1:4173'

export default defineConfig({
	testDir: './tests/e2e',
	fullyParallel: false,
	forbidOnly: !!process.env.CI,
	retries: process.env.CI ? 2 : 0,
	workers: 1,
	reporter: 'list',
	use: {
		baseURL: process.env.E2E_BASE_URL || 'http://127.0.0.1:4173',
		trace: 'on-first-retry',
	},
	projects: [
		{
			name: 'api',
			testMatch: /.*\.spec\.ts/,
		},
	],
	webServer: !process.env.E2E_BASE_URL
		? {
				command: 'pnpm dev',
				url: 'http://127.0.0.1:4173',
				timeout: 120000,
				reuseExistingServer: true,
				env: {
					DEV_PROXY: 'http://192.168.1.135:19876',
				},
			}
		: undefined,
})
