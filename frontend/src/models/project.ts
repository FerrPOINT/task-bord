// @ts-nocheck
import AbstractModel from './abstractModel'
import UserModel from '@/models/user'

import type {IProject} from '@/modelTypes/IProject'
import type {IUser} from '@/modelTypes/IUser'

export default class ProjectModel extends AbstractModel<IProject> implements IProject {
	id = 0
	title = ''
	description = ''
	owner: IUser | null = null
	ownerId = 0
	isArchived = false
	hexColor = ''
	identifier = ''
	parentProjectId = 0
	created: Date = null
	updated: Date = null

	constructor(data: Partial<IProject> = {}) {
		super()
		this.assignData(data)

		if (this.owner) {
			this.owner = new UserModel(this.owner)
		}

		if (this.hexColor !== '' && this.hexColor.substring(0, 1) !== '#') {
			this.hexColor = '#' + this.hexColor
		}

		this.created = this.created ? new Date(this.created) : null
		this.updated = this.updated ? new Date(this.updated) : null
	}
}
