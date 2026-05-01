<script setup lang="ts">
import { ref, watchEffect, type Component } from 'vue'

const props = defineProps<{
  /** Lucide icon name in kebab-case (e.g. "wrench", "arrow-up"). Strip "lucide:" prefix before passing. */
  name: string
  /** Pixel size for width/height (default 24). */
  size?: number | string
  /** Stroke width override (default 2). */
  strokeWidth?: number
}>()

// Module-level cache: the dynamic import resolves once per page load.
// All <LucideIcon> instances share the same loaded namespace.
let cached: Record<string, Component> | null = null
let loadingPromise: Promise<Record<string, Component>> | null = null

async function loadLucideModule(): Promise<Record<string, Component>> {
  if (cached) return cached
  if (!loadingPromise) {
    loadingPromise = import('lucide-vue-next').then((mod) => {
      cached = mod as unknown as Record<string, Component>
      return cached
    })
  }
  return loadingPromise
}

function toPascalCase(s: string): string {
  return s
    .split(/[-_]/)
    .filter(Boolean)
    .map((w) => w.charAt(0).toUpperCase() + w.slice(1).toLowerCase())
    .join('')
}

const Icon = ref<Component | null>(null)
const state = ref<'loading' | 'loaded' | 'missing'>('loading')

watchEffect(async () => {
  state.value = 'loading'
  Icon.value = null
  const trimmed = props.name.trim()
  if (!trimmed) {
    state.value = 'missing'
    return
  }
  const mod = await loadLucideModule()
  const pascal = toPascalCase(trimmed)
  const found = mod[pascal]
  if (found) {
    Icon.value = found
    state.value = 'loaded'
  } else {
    state.value = 'missing'
  }
})
</script>

<template>
  <component
    v-if="state === 'loaded' && Icon"
    :is="Icon"
    :size="size ?? 24"
    :stroke-width="strokeWidth ?? 2"
    class="lucide-icon"
    :data-lucide-loaded="name"
  />
  <span
    v-else-if="state === 'loading'"
    class="lucide-icon lucide-icon--loading"
    data-lucide-loading="true"
    :style="{ width: `${size ?? 24}px`, height: `${size ?? 24}px` }"
    aria-hidden="true"
  />
  <span
    v-else
    class="lucide-icon lucide-icon--missing"
    :data-lucide-missing="name"
    :title="`Lucide icon not found: ${name}`"
    :style="{ width: `${size ?? 24}px`, height: `${size ?? 24}px` }"
  >?</span>
</template>

<style scoped>
.lucide-icon {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  flex-shrink: 0;
}
.lucide-icon--loading {
  /* Transparent while async chunk loads — avoids visual jitter when icon
     suddenly appears. e2e tests wait for [data-lucide-loading] to disappear. */
}
.lucide-icon--missing {
  font-size: 0.85em;
  font-weight: 600;
  color: rgba(255, 193, 77, 0.7);
}
</style>
