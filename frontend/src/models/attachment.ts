// @ts-nocheck
import AbstractModel from './abstractModel'
import UserModel from './user'

import type { IUser } from '@/modelTypes/IUser'
import type { IAttachment } from '@/modelTypes/IAttachment'

export const SUPPORTED_IMAGE_SUFFIX = ['.jpeg', '.jpg', '.png', '.bmp', '.gif']
export const SUPPORTED_PDF_SUFFIX = ['.pdf']

export function canPreviewImage(attachment: IAttachment): boolean {
	return SUPPORTED_IMAGE_SUFFIX.some((suffix) => attachment.file.name.toLowerCase().endsWith(suffix))
}

export function canPreviewPdf(attachment: IAttachment): boolean {
	return SUPPORTED_PDF_SUFFIX.some((suffix) => attachment.file.name.toLowerCase().endsWith(suffix))
}

export function canPreview(attachment: IAttachment): boolean {
	return canPreviewImage(attachment) || canPreviewPdf(attachment)
}

export default class AttachmentModel extends AbstractModel<IAttachment> implements IAttachment {
	id = 0
	taskId = 0
	createdBy: IUser | null = null
	file: IAttachment['file'] = {id: 0, name: '', size: 0, mime: ''}
	created: Date = null

	constructor(data: Partial<IAttachment>) {
		super()
		this.assignData(data)

		if (this.createdBy) {
			this.createdBy = new UserModel(this.createdBy)
		}
		if (!this.file) {
			this.file = {id: 0, name: '', size: 0, mime: ''}
		}
		if (this.created) {
			this.created = new Date(this.created)
		}
	}
}
