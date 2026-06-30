import type {IUser} from './IUser'

export interface IProject {
	id: number
	title: string
	description?: string
	identifier?: string
	hexColor?: string
	ownerId?: number
	owner?: IUser | null
	parentProjectId?: number
	isArchived?: boolean
	created?: string
	updated?: string
	maxPermission?: number
}
