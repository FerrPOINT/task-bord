<script lang="ts" setup>
// @ts-nocheck

import { computed, onMounted, ref } from 'vue'
import { useBaseStore } from '@/stores/base'
import { useTaskStore } from '@/stores/tasks'
import TaskService from '@/services/task'

const baseStore = useBaseStore()
const taskStore = useTaskStore()

const hasTasks = computed(() => baseStore.hasTasks)
const loading = computed(() => taskStore.isLoading)
const show = ref(false)

onMounted(async () => {
	show.value = false

	if (hasTasks.value) {
		show.value = false
		return
	}

	const taskService = new TaskService()
	const tasks = await taskService.getAll(undefined, {per_page: 1})
	show.value = tasks.length === 0
})
</script>

<template>
	<template v-if="show && !loading">
		<p class="mbs-4">
			{{ $t('home.project.importText') }}
		</p>
	</template>
</template>
