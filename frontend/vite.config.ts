/// <reference types="vitest" />
import {defineConfig, type PluginOption, loadEnv} from 'vite'
import {configDefaults} from 'vitest/config'
import vue from '@vitejs/plugin-vue'
import {URL, fileURLToPath} from 'node:url'
import {dirname, resolve} from 'node:path'
import {readFileSync} from 'node:fs'

import VueI18nPlugin from '@intlify/unplugin-vue-i18n/vite'
import UnpluginInjectPreload from 'unplugin-inject-preload/vite'

import svgLoader from 'vite-svg-loader'
import postcssPresetEnv from 'postcss-preset-env'
import postcssEasingGradients from 'postcss-easing-gradients'
import tailwindcss from '@tailwindcss/vite'
import vueDevTools from 'vite-plugin-vue-devtools'

const pathSrc = fileURLToPath(new URL('./src', import.meta.url)).replaceAll('\\', '/')

// the @use rules have to be the first in the compiled stylesheets
const PREFIXED_SCSS_STYLES = `@use "sass:math";
@import "${pathSrc}/styles/common-imports.scss";`

/**
 * @param fontNames Array of the file names of the fonts without axis and hash suffixes
 */
function createFontMatcher(fontNames: string[]) {
	// The `match` option for the files of VitePluginInjectPreload
	// matches the _output_ files.
	// Since we only want to mach variable fonts, we exploit here the fact
	// that we added the `wght` term to indicate the variable weight axis.
	// The format is something like:
	// `/assets/OpenSans-Italic_wght__c9a8fe68-5f21f1e7.woff2`
	// see: https://regex101.com/r/UgUWr1/1
	return new RegExp(`^.+\\/(${fontNames.join('|')})_wght__[a-z1-9]{8}-[a-z1-9]{8}\\.woff2$`)
}

// https://vitejs.dev/config/
export default defineConfig(({command, mode}) => {
	// Load env file based on `mode` in the current working directory.
	// Set the third parameter to '' to load all env regardless of the `VITE_` prefix.
	// https://vitejs.dev/config/#environment-variables
	const env = loadEnv(mode, process.cwd(), '')

	switch (command) {
		case 'serve':
			// this is DEV mode 
			return getServeConfig(env)
			// return getBuildConfig(env)
		case 'build':
			// build for prodution
			return getBuildConfig(env)
	}
})

function getBuildConfig(env: Record<string, string>) {
	return {
		base: env.TASKBOARD_FRONTEND_BASE,
		define: {},
		// https://vitest.dev/config/
		test: {
			environment: 'happy-dom',
			exclude: [...configDefaults.exclude, 'e2e/**'],
			'vitest.commandLine': 'pnpm test:unit',
		},
		css: {
			preprocessorOptions: {
				sass: {
					quietDeps: true, // silence deprecation warnings
				},
				scss: {
					additionalData: PREFIXED_SCSS_STYLES,
					charset: false, // fixes  "@charset" must be the first rule in the file" warnings,
					quietDeps: true, // silence deprecation warnings
				},
			},
			postcss: {
				plugins: [
					postcssEasingGradients(),
					postcssPresetEnv({
						features: {
							'logical-properties-and-values': false,
						}
					}),
				],
			},
		},
		plugins: [
			tailwindcss(),
			vue(),
			svgLoader({
				// Since the svgs are already manually optimized via https://jakearchibald.github.io/svgomg/
				// we don't need to optimize them again.
				svgo: false,
			}),
			VueI18nPlugin({
				// TODO: only install needed stuff
				// Whether to install the full set of APIs, components, etc. provided by Vue I18n.
				// By default, all of them will be installed.
				fullInstall: true,
				include: resolve(dirname(pathSrc), './src/i18n/lang/**'),
			}),
			// https://github.com/Applelo/unplugin-inject-preload
			UnpluginInjectPreload({
				files: [{
					outputMatch: createFontMatcher(['Quicksand', 'OpenSans', 'OpenSans-Italic']),
					attributes: {crossorigin: 'anonymous'},
				}],
				injectTo: 'custom',
			}),
			vueDevTools({
				launchEditor: env.VUE_DEVTOOLS_LAUNCH_EDITOR || 'code',
			}),
		],
		resolve: {
			alias: [
				{
					find: '@',
					replacement: pathSrc,
				},
			],
			extensions: ['.mjs', '.js', '.ts', '.jsx', '.tsx', '.json', '.vue'],
		},
		server: {
			host: '127.0.0.1', // see: https://github.com/vitejs/vite/pull/8543
			port: parseInt(env.TASKBOARD_FRONTEND_PORT || '4173', 10),
			strictPort: true,
		},
		output: {
			manualChunks: {
			},
		},
		build: {
			target: 'esnext',
			rollupOptions: {
				plugins: [
				],
			},
		},
	}
}

function getServeConfig(env: Record<string, string>) {
	// get some default settings from prod mod
	const buildConfig = getBuildConfig(env)

	// Build the proxy pattern from TASKBOARD_FRONTEND_BASE so that custom base
	// paths like /task-board proxy /task-board/api/* correctly.
	// Falls back to /api.
	const base = (env.TASKBOARD_FRONTEND_BASE || '/').replace(/\/+$/, '')
	const proxyPath = `${base}/api`

	const proxy = env.DEV_PROXY ? {
		[proxyPath]: {
			target: env.DEV_PROXY,
			changeOrigin: true,
			secure: false,
			// Strips prefix for the backend
			rewrite: (path: string) => path.replace(new RegExp(`^${base.replace(/[.*+?^${}()|[\]\\]/g, '\\$&')}`), ''),
		},
	} : undefined

	// override prod settings with dev settings
	return {
		...buildConfig,
		server: {
			...buildConfig.server,
			...(proxy && {proxy}),
		},
		preview: {
			host: '127.0.0.1',
			port: parseInt(env.TASKBOARD_FRONTEND_PORT || '4173', 10),
			strictPort: true,
			...(proxy && {proxy}),
		},
	}
}
