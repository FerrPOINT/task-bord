// @ts-nocheck
import type {Prefixes} from './types'

const TASK_BOARD_PREFIXES: Prefixes = {
	label: '*',
	project: '+',
	priority: '!',
	assignee: '@',
}

export enum PrefixMode {
	Disabled = 'disabled',
	Default = 'task-board',
}

export const PREFIXES = {
	[PrefixMode.Disabled]: undefined,
	[PrefixMode.Default]: TASK_BOARD_PREFIXES,
}
