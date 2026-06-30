<template>
	<a href="#main-content" class="skip-to-content">
		{{ $t('misc.skipToContent') }}
	</a>
	<template v-if="showAuthLayout">
		<AppHeader />
		<ContentAuth />
	</template>
	<NoAuthWrapper v-else show-api-config>
		<RouterView />
	</NoAuthWrapper>
	<Teleport to="body">
		<DemoMode />
	</Teleport>
</template>

<script lang="ts" setup>
// @ts-nocheck
import {computed} from 'vue'
import {useRoute} from 'vue-router'
import isTouchDevice from 'is-touch-device'

import AppHeader from '@/components/home/AppHeader.vue'
import ContentAuth from '@/components/home/ContentAuth.vue'
import NoAuthWrapper from '@/components/misc/NoAuthWrapper.vue'

import {DEFAULT_LANGUAGE, setLanguage} from '@/i18n'

import {useAuthStore} from '@/stores/auth'

import {useColorScheme} from '@/composables/useColorScheme'
import {useBodyClass} from '@/composables/useBodyClass'
import DemoMode from '@/components/home/DemoMode.vue'

const authStore = useAuthStore()

const PUBLIC_ROUTE_NAMES = new Set([
	'user.login',
	'user.register',
])

const route = useRoute()
const showAuthLayout = computed(() => authStore.authUser && typeof route.name === 'string' && !PUBLIC_ROUTE_NAMES.has(route.name))

useBodyClass('is-touch', isTouchDevice())

setLanguage(DEFAULT_LANGUAGE)
useColorScheme()
</script>

<style src="@/styles/tailwind.css" />

<style lang="scss" src="@/styles/global.scss" />
