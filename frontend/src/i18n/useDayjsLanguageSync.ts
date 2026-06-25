// @ts-nocheck
import type dayjs from 'dayjs'
import { computed, ref, watch } from 'vue'

import { i18n, type ISOLanguage, type SupportedLocale } from '@/i18n'

export const DAYJS_LOCALE_MAPPING = {
	'ru-ru': 'ru',
	'en': 'en',
} as Record<SupportedLocale, ISOLanguage>

export const DAYJS_LANGUAGE_IMPORTS = {
	'ru-ru': () => import('dayjs/locale/ru'),
} as Record<SupportedLocale, () => Promise<ILocale>>

export async function loadDayJsLocale(language: SupportedLocale) {
	if (language === 'en') {
		return
	}

	await DAYJS_LANGUAGE_IMPORTS[language.toLowerCase()]()
}

export function useDayjsLanguageSync(dayjsGlobal: typeof dayjs) {

	const dayjsLanguageLoaded = ref(false)
	watch(
		() => i18n.global.locale.value,
		async (currentLanguage: string) => {
			if (!dayjsGlobal) {
				return
			}
			const dayjsLanguageCode = DAYJS_LOCALE_MAPPING[currentLanguage.toLowerCase()] || currentLanguage.toLowerCase()
			dayjsLanguageLoaded.value = dayjsGlobal.locale() === dayjsLanguageCode
			if (dayjsLanguageLoaded.value) {
				return
			}
			await loadDayJsLocale(currentLanguage)
			dayjsGlobal.locale(dayjsLanguageCode)
			dayjsLanguageLoaded.value = true
		},
		{immediate: true},
	)

	// we export the loading state since that's easier to work with
	const isLoading = computed(() => !dayjsLanguageLoaded.value)

	return isLoading
}
