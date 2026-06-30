// @ts-nocheck
import AbstractService from './abstractService'
import ProjectModel from '@/models/project'
import type {IProject} from '@/modelTypes/IProject'
import {colorFromHex} from '@/helpers/color/colorFromHex'

export default class ProjectService extends AbstractService<IProject> {
	constructor() {
		super({
			create: '/projects',
			get: '/projects/{id}',
			getAll: '/projects',
			update: '/projects/{id}',
			delete: '/projects/{id}',
		})
	}

	modelFactory(data) {
		return new ProjectModel(data)
	}

	beforeUpdate(model) {
		if(typeof model.hexColor !== 'undefined') {
			model.hexColor = colorFromHex(model.hexColor)
		}
		return model
	}

	beforeCreate(project) {
		project.hexColor = colorFromHex(project.hexColor)
		return project
	}
}
