export interface ITask {
	id: number
	title: string
	description?: string
	done?: boolean
	doneAt?: string | null
	dueDate?: string | null
	projectId: number
	index?: number
	identifier?: string
	created?: string
	updated?: string
	createdBy?: IUser | null
	createdById?: number
	assignees?: IUser[]
	labels?: ILabel[]
	comments?: ITaskComment[]
	commentCount?: number
	maxPermission?: number
}
