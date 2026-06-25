<script setup lang="ts">
// @ts-nocheck



import Logo from '@/assets/logo.svg?component'

const CustomLogo = window.CUSTOM_LOGO_URL || ''
</script>

<template>
	<div class="logo-wrapper">
		<Logo
			v-if="!CustomLogo"
			alt="Task Board"
			class="logo-icon"
		/>
		<img
			v-show="CustomLogo"
			:src="CustomLogo"
			alt="Task Board"
			class="logo-icon"
		>
		<span class="logo-text">Task Board</span>
	</div>
</template>

<style lang="scss" scoped>
.logo-wrapper {
	display: flex;
	align-items: center;
	gap: 0.75rem;
}

.logo-icon {
	max-inline-size: 48px;
	max-block-size: 48px;
}

.logo-text {
	font-family: system-ui, -apple-system, sans-serif;
	font-size: 1.5rem;
	font-weight: 700;
	color: #ffffff;
	white-space: nowrap;
}
</style>
