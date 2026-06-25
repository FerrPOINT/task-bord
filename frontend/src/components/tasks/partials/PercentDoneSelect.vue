<template>
	<div class="select">
		<select
			v-model.number="percentDone"
			:disabled="disabled || undefined"
		>
			<option
				v-for="option in PERCENT_OPTIONS"
				:key="option.value"
				:value="option.value"
			>
				{{ option.label }}
			</option>
		</select>
	</div>
</template>

<script setup lang="ts">
// @ts-nocheck



withDefaults(defineProps<{
	disabled?: boolean
}>(), {
	disabled: false,
})

const percentDone = defineModel<number>({ required: true })

const PERCENT_OPTIONS = [
	{ value: 0, label: '0% — не начато' },
	{ value: 0.1, label: '10%' },
	{ value: 0.2, label: '20%' },
	{ value: 0.3, label: '30%' },
	{ value: 0.4, label: '40%' },
	{ value: 0.5, label: '50%' },
	{ value: 0.6, label: '60%' },
	{ value: 0.7, label: '70%' },
	{ value: 0.8, label: '80%' },
	{ value: 0.9, label: '90%' },
	{ value: 1, label: '100% — готово' },
] as const
</script>
