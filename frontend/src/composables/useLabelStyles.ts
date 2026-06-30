// @ts-nocheck
import {colorFromHex} from '@/helpers/color/colorFromHex'
import {getTextColor} from '@/helpers/color/getTextColor'
import type {ILabel} from '@/modelTypes/ILabel'

export function useLabelStyles() {
	function getLabelStyles(label: ILabel) {
		const color = colorFromHex(label.hexColor || '#ffffff')
		const textColor = getTextColor(color)
		return {
			backgroundColor: color,
			color: textColor,
		}
	}
	return {getLabelStyles}
}
