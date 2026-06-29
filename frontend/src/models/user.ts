import AbstractModel from './abstractModel'

import {AUTH_TYPES, type IUser, type AuthType} from '@/modelTypes/IUser'

export function getDisplayName(user: IUser) {
	if (user.name !== '') {
		return user.name
	}

	return user.username
}

export default class UserModel extends AbstractModel<IUser> implements IUser {
	id = 0
	email = ''
	username = ''
	name = ''
	exp = 0
	type: AuthType = AUTH_TYPES.UNKNOWN

	created: Date
	updated: Date

	isLocalUser: boolean
	deletionScheduledAt: null
	isAdmin?: boolean
	botOwnerId = 0

	constructor(data: Partial<IUser> = {}) {
		super()
		this.assignData(data)

		this.created = new Date(this.created)
		this.updated = new Date(this.updated)
	}

	get isBot(): boolean {
		return (this.botOwnerId ?? 0) > 0
	}
}
