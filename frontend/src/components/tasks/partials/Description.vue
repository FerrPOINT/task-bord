<template>
	<div
		:class="{'d-print-none': isEmpty}"
		class="task-section"
	>
		<h3>
			<span class="icon is-grey">
				<Icon icon="align-left" />
			</span>
			{{ $t('task.attributes.description') }}
			<CustomTransition name="fade">
				<span
					v-if="loading && saving"
					class="is-small is-inline-flex"
				>
					<span class="loader is-inline-block mie-2" />
					{{ $t('misc.saving') }}
				</span>
				<span
					v-else-if="!loading && saved"
					class="is-small has-text-success"
				>
					<Icon icon="check" />
					{{ $t('misc.saved') }}
				</span>
			</CustomTransition>
		</h3>
		<div class="description-editor">
			<Editor
				v-model="description"
				class="tiptap__task-description"
				:is-edit-enabled="canWrite"
				:upload-callback="uploadCallback"
				:placeholder="$t('task.description.placeholder')"
				:show-save="true"
				edit-shortcut="KeyE"
				:enable-discard-shortcut="true"
				:enable-mentions="true"
				:mention-project-id="modelValue.projectId"
				:storage-key="descriptionStorageKey"
				@update:modelValue="saveWithDelay"
				@save="save"
			/>
		</div>
	</div>
</template>

<script setup lang="ts">
// @ts-nocheck



import {ref, computed, watchEffect,  onBeforeUnmount} from 'vue'
import {onBeforeRouteLeave} from 'vue-router'

import CustomTransition from '@/components/misc/CustomTransition.vue'
import Editor from '@/components/input/AsyncEditor'

import { clearEditorDraft } from '@/helpers/editorDraftStorage'
import { isEditorContentEmpty } from '@/helpers/editorContentEmpty'
import type { ITask } from '@/modelTypes/ITask'
import { useTaskStore } from '@/stores/tasks'

export type AttachmentUploadFunction = (file: File, onSuccess: (attachmentUrl: string) => void) => Promise<string>

const props = defineProps<{
	modelValue: ITask,
	attachmentUpload: AttachmentUploadFunction,
	canWrite: boolean,
}>()

const emit = defineEmits<{
	'update:modelValue': [value: ITask]
}>()

const description = ref<string>('')
const hasChanges = ref(false)
watchEffect(() => {
	description.value = props.modelValue.description
	hasChanges.value = false
})

const saved = ref(false)

// Since loading is global state, this variable ensures we're only showing the saving icon when saving the description.
const saving = ref(false)

const taskStore = useTaskStore()
const loading = computed(() => taskStore.isLoading)

const changeTimeout = ref<ReturnType<typeof setTimeout> | null>(null)

const descriptionStorageKey = computed(() => `task-description-${props.modelValue.id}`)

const isEmpty = computed(() => isEditorContentEmpty(description.value))

async function saveWithDelay() {
	if (description.value === props.modelValue.description) {
		hasChanges.value = false
		if (changeTimeout.value !== null) {
			clearTimeout(changeTimeout.value)
		}
		return
	}

	hasChanges.value = true
	if (changeTimeout.value !== null) {
		clearTimeout(changeTimeout.value)
	}

	changeTimeout.value = setTimeout(async () => {
		await save()
	}, 5000)
}

onBeforeUnmount(async () => {
	await save() // Save before unmounting to handle modal race condition
	if (changeTimeout.value !== null) {
		clearTimeout(changeTimeout.value)
	}
})

onBeforeRouteLeave(() => save())

async function save() {
	if (!hasChanges.value) {
		return
	}

	hasChanges.value = false
	if (changeTimeout.value !== null) {
		clearTimeout(changeTimeout.value)
	}
	saved.value = false
	saving.value = true

	try {
		const updated = await taskStore.update({
			...props.modelValue,
			description: description.value,
		})
		emit('update:modelValue', updated)

		// Clear draft from localStorage when saved successfully
		clearEditorDraft(descriptionStorageKey.value)

		saved.value = true
		setTimeout(() => {
			saved.value = false
		}, 2000)
	} catch (error) {
		// If the task was deleted (404), silently skip saving
		if (error?.response?.status === 404) {
			return
		}
		hasChanges.value = true
		// Re-throw other errors
		throw error
	} finally {
		saving.value = false
	}
}

async function uploadCallback(files: File[] | FileList): Promise<string[]> {
	const uploadPromises: Promise<string>[] = []

	files.forEach((file: File) => {
		const promise = new Promise<string>((resolve) => {
			props.attachmentUpload(file, (uploadedFileUrl: string) => resolve(uploadedFileUrl))
		})

		uploadPromises.push(promise)
	})

	return await Promise.all(uploadPromises)
}
</script>

<style lang="scss" scoped>
.tiptap__task-description {
	margin-inline-start: 0;
}

.description-editor {
	background: transparent;
	border: 0;
	border-radius: 0;
	padding: 0;

	:deep(.tiptap__editor) {
		border: 0;
		background: transparent;

		&.tiptap__editor-is-edit-enabled {
			background: var(--scheme-main);
			box-shadow: none;
		}
	}
}
</style>
