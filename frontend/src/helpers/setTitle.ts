// @ts-nocheck
export function setTitle(title : undefined | string) {
	document.title = (typeof title === 'undefined' || title === '')
		? 'Task Board'
		: `${title} | Task Board`
}
