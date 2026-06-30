<template>
	<NodeViewWrapper class="mention-user">
		<span class="mention__label">
			{{ node.attrs.label ?? node.attrs.id }}
		</span>
	</NodeViewWrapper>
</template>

<script lang="ts" setup>
// @ts-nocheck
import {nodeViewProps, NodeViewWrapper} from '@tiptap/vue-3'

const props = defineProps(nodeViewProps)
</script>

<style lang="scss">
.tiptap .mention-user {
    display: inline-flex;
    align-items: center;
    position: relative;
    inset-block-end: 0;
    padding-inline-start: 0.25rem;
}
</style>
