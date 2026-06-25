<template>
	<div class="edit-assignees">
		<AssigneeList
			v-if="assignees.length > 0"
			:assignees="assignees"
			:disabled="disabled"
			can-remove
			inline
			@remove="removeAssignee"
		/>
		<BaseButton
			v-if="!disabled && !isOpen"
			class="add-assignee-button"
			@click="openAssigneeInput"
		>
			<span class="icon is-small">
				<Icon icon="plus" />
			</span>
			<span v-if="assignees.length === 0">{{ $t('task.assignee.add') }}</span>
		</BaseButton>
		<Multiselect
			v-if="isOpen"
			ref="multiselectRef"
			v-model="assignees"
			class="edit-assignees__select"
			:loading="projectUserService.loading"
			:placeholder="$t('task.assignee.placeholder')"
			:multiple="true"
			:search-results="foundUsers"
			label="name"
			:select-placeholder="$t('task.assignee.selectPlaceholder')"
			:autocomplete-enabled="false"
			@search="findUser"
			@select="addAssignee"
		>
			<template #searchResult="{option: user}">
				<User
					:avatar-size="24"
					:show-username="true"
					:user="user"
				/>
			</template>
		</Multiselect>
	</div>
</template>

<script setup lang="ts">
// @ts-nocheck



import {ref, shallowReactive, watch, nextTick, onMounted, onBeforeUnmount} from 'vue'
import {useI18n} from 'vue-i18n'

import User from '@/components/misc/User.vue'
import Multiselect from '@/components/input/Multiselect.vue'
import BaseButton from '@/components/base/BaseButton.vue'
import Icon from '@/components/misc/Icon'

import {includesById} from '@/helpers/utils'
import ProjectUserService from '@/services/projectUsers'
import {success} from '@/message'
import {useTaskStore} from '@/stores/tasks'

import type {IUser} from '@/modelTypes/IUser'
import {getDisplayName} from '@/models/user'
import AssigneeList from '@/components/tasks/partials/AssigneeList.vue'

const props = withDefaults(defineProps<{
	modelValue: IUser[] | undefined,
	taskId: number,
	projectId: number,
	disabled?: boolean,
}>(), {
	disabled: false,
})

const emit = defineEmits<{
	'update:modelValue': [value: IUser[] | undefined],
}>()

const taskStore = useTaskStore()
const {t} = useI18n({useScope: 'global'})

const projectUserService = shallowReactive(new ProjectUserService())
const foundUsers = ref<IUser[]>([])
const assignees = ref<IUser[]>([])
const isOpen = ref(false)
const multiselectRef = ref<InstanceType<typeof Multiselect> | null>(null)
let isAdding = false

watch(
	() => props.modelValue,
	(value) => {
		assignees.value = value
		if (value && value.length > 0) {
			isOpen.value = false
		}
	},
	{
		immediate: true,
		deep: true,
	},
)

function openAssigneeInput() {
	isOpen.value = true
	nextTick(() => {
		multiselectRef.value?.$el?.querySelector('input')?.focus()
		findUser('')
	})
}

function closeAssigneeInputIfEmpty() {
	if (assignees.value.length === 0) {
		isOpen.value = false
	}
}

function handleClickOutside(event: MouseEvent) {
	const root = (multiselectRef.value?.$el as HTMLElement | undefined)?.closest('.edit-assignees')
	if (isOpen.value && root && !root.contains(event.target as Node)) {
		closeAssigneeInputIfEmpty()
	}
}

onMounted(() => document.addEventListener('click', handleClickOutside))
onBeforeUnmount(() => document.removeEventListener('click', handleClickOutside))

async function addAssignee(user: IUser) {
	if (isAdding) {
		return
	}

	try {
		nextTick(() => isAdding = true)
		await taskStore.addAssignee({user: user, taskId: props.taskId})
		emit('update:modelValue', assignees.value)
		success({message: t('task.assignee.assignSuccess')})
	} finally {
		nextTick(() => isAdding = false)
		isOpen.value = false
	}
}

async function removeAssignee(user: IUser) {
	await taskStore.removeAssignee({user: user, taskId: props.taskId})

	// Remove the assignee from the project
	const idx = assignees.value.findIndex(a => a.id === user.id)
	if (idx !== -1) {
		assignees.value.splice(idx, 1)
	}
	success({message: t('task.assignee.unassignSuccess')})
}

async function findUser(query: string) {
	const response = await projectUserService.getAll({projectId: props.projectId}, {s: query}) as IUser[]

	// Filter the results to not include users who are already assigned
	foundUsers.value = response
		.filter(({id}) => !includesById(assignees.value, id))
		.map(u => {
			// Users may not have a display name set, so we fall back on the username in that case
			u.name = getDisplayName(u)
			return u
		})
}
</script>

<style lang="scss">
.edit-assignees {
	display: flex;
	align-items: center;
	gap: .5rem;
	flex-wrap: wrap;

	.add-assignee-button {
		display: inline-flex;
		align-items: center;
		gap: .25rem;
		padding: .25rem .5rem;
		border-radius: $radius;
		background: var(--scheme-main-bis);
		color: var(--text);
		font-size: .875rem;
		transition: background $transition;

		&:hover {
			background: var(--grey-200);
		}
	}

	.edit-assignees__select {
		flex: 1;
		min-inline-size: 200px;
	}
}

.edit-assignees.has-assignees.multiselect .input {
	padding-inline-start: 0;
}
</style>
