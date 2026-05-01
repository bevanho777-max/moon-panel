<script setup lang="ts">
import { NConfigProvider, NMessageProvider, darkTheme } from 'naive-ui'
import { computed, onMounted, watch } from 'vue'
import { useAuthStore } from './stores/auth'
import { useUIStore } from './stores/ui'

const auth = useAuthStore()
const ui = useUIStore()

// Phase 1: dark by default. Phase 2.5c adds primary-color override on top —
// the rest of darkTheme stays untouched, so disabling the override (theme
// primary = null) returns to the exact previous look.
const theme = computed(() => darkTheme)
const themeOverrides = computed(() => ui.themeOverrides)

// Phase 2.5c-acrylic: body class toggle for the global acrylic CSS rules
// in main.css. When wallpaper is set, body.has-wallpaper activates the
// frosted-glass surfaces on .n-card / .mp-acrylic-* elements; when null,
// surfaces fall back to NaiveUI's solid dark theme defaults.
//
// Watching ui.wallpaper directly (not a `hasWallpaper` computed) keeps the
// dependency one step shallower — the body classList side effect doesn't
// need its own derived state. immediate:true so the class reflects the
// loaded value on first paint after ui.ensureLoaded resolves.
watch(
  () => ui.wallpaper,
  (val) => {
    document.body.classList.toggle('has-wallpaper', !!val)
  },
  { immediate: true },
)

onMounted(() => {
  auth.refresh()
  // Wallpaper / theme load runs in parallel with auth — neither blocks the
  // other. Errors inside ensureLoaded fall back to defaults silently
  // (private-mode login page would 401 here; that's expected).
  ui.ensureLoaded()
})
</script>

<template>
  <NConfigProvider :theme="theme" :theme-overrides="themeOverrides">
    <!-- Wallpaper layer: fixed full-viewport, behind everything (z-index:-1).
         CSS `filter: blur()` is on this layer alone — front-of-screen content
         is never blurred. `transform: translateZ(0)` forces a GPU compositor
         layer so blur stays smooth on mobile / low-end devices. -->
    <div
      v-if="ui.wallpaperUrl"
      class="wallpaper-layer"
      :style="{
        backgroundImage: `url(${ui.wallpaperUrl})`,
        filter: ui.blur > 0 ? `blur(${ui.blur}px)` : 'none',
      }"
      aria-hidden="true"
    />
    <NMessageProvider>
      <router-view />
    </NMessageProvider>
  </NConfigProvider>
</template>

<style>
.wallpaper-layer {
  position: fixed;
  inset: 0;
  z-index: -1;
  background-size: cover;
  background-position: center center;
  background-repeat: no-repeat;
  pointer-events: none;
  /* GPU compositor layer for smooth blur, even on mobile. */
  transform: translateZ(0);
  will-change: filter;
  /* When blur > 0, blurred edges expose the underlying body color. Scaling
     up by ~1.05 hides the soft edge without distorting the visible center. */
}
.wallpaper-layer[style*="blur"] {
  transform: translateZ(0) scale(1.05);
}
</style>
