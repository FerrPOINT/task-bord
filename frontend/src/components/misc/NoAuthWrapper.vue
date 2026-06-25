<template>
	<div class="no-auth-wrapper">
		<slot />
		<ApiConfig
			v-if="showApiConfig"
			:configure-open="configureOpen"
		/>
	</div>
</template>

<script lang="ts" setup>
// @ts-nocheck

defineProps({
	showApiConfig: {
		type: Boolean,
		default: false,
	},
	configureOpen: {
		type: Boolean,
		default: false,
	},
})
</script>
