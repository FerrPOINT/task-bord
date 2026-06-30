export interface IAttachment {
	id: number
	taskId: number
	created: string
	file: {
		id: number
		name: string
		size: number
		mime: string
	}
	createdById?: number
	createdBy?: import('./IUser').IUser | null
}
