<script setup lang="ts">
import type { HighlightSegment } from '@/utils/cardSearch'

/**
 * Renders a pre-split segment array with matched runs wrapped in <mark>.
 * Segments carry plain strings and are rendered as text nodes (no v-html), so
 * card titles containing markup stay inert.
 */
defineProps<{ segments: HighlightSegment[] }>()
</script>

<template><template v-for="(seg, i) in segments" :key="i"><mark v-if="seg.mark" class="hl">{{ seg.text }}</mark><template v-else>{{ seg.text }}</template></template></template>

<style scoped>
.hl {
  background: var(--mp-search-mark-bg, rgba(255, 214, 102, 0.32));
  color: inherit;
  border-radius: 2px;
  padding: 0 1px;
}
</style>
