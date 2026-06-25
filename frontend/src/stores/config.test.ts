// @ts-nocheck
import {describe, it, expect, beforeEach} from 'vitest'
import {setActivePinia, createPinia} from 'pinia'

import {useConfigStore} from './config'

describe('config store', () => {
	beforeEach(() => {
		setActivePinia(createPinia())
	})

	it('holds server config', () => {
		const store = useConfigStore()
		expect(store).toBeDefined()
	})
})
