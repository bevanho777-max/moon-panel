<script setup lang="ts">
import { NConfigProvider, NMessageProvider, darkTheme } from 'naive-ui'
import { computed, onMounted, watch } from 'vue'
import { useRoute } from 'vue-router'
import StarfieldBackground from './components/StarfieldBackground.vue'
import { useAuthStore } from './stores/auth'
import { useUIStore } from './stores/ui'

const auth = useAuthStore()
const ui = useUIStore()
const route = useRoute()

// v0.2.25: gate the dynamic background on non-admin routes. Admin's
// drag-heavy UX (Cards reorder) is the existing INP hotspot — keeping
// the animated layer off /admin/* means the existing acrylic-on-modals
// path stays the only backdrop-filter cost there.
const isAdminRoute = computed(() => route.path.startsWith('/admin'))

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

// v0.2.0: keep the browser tab title in sync with the admin-set siteTitle.
// Runs on every change (including the initial hydration from getPanel),
// so a fresh page load shows "Foo Lab" immediately rather than briefly
// flashing "Moon Panel".
watch(
  () => ui.siteTitle,
  (val) => {
    if (val) document.title = val
  },
  { immediate: true },
)

// v0.2.1: theme preset → <html data-theme="..."> + lazy-load risen-only
// fonts. CRITICAL: moon (the default) must trigger NO font network
// requests, so the dynamic import sits inside an `if (val === 'risen')`
// branch. Once loaded the fonts stay cached for the session; switching
// back to moon doesn't unload them but `--mp-brand-font` for moon points
// at the system stack so they're not used.
const fontsLoaded = { risen: false }
watch(
  () => ui.themePreset,
  async (val) => {
    document.documentElement.dataset.theme = val
    if (val === 'risen' && !fontsLoaded.risen) {
      fontsLoaded.risen = true
      try {
        await Promise.all([
          import('@fontsource/playfair-display/700.css'),
          import('@fontsource/noto-serif-sc/700.css'),
        ])
      } catch {
        // Font CDN / network failure: serif fallbacks in --mp-brand-font
        // (Georgia / SimSun) keep the look approximately correct. Don't
        // surface a user-visible error for this.
      }
    }
  },
  { immediate: true },
)

// Private-mode hydration: when MOON_PUBLIC_MODE=false, /api/public/panel is
// gated behind auth, so the cold-start ui.ensureLoaded() in onMounted hits
// a 401 and silently bails — leaving builtins/wallpaper/theme empty and
// breaking the admin WallpaperPicker grid (v0.1.0 bug). Once the user
// flips to authenticated (login / TOTP / first-time admin init all land
// here), re-pull the panel so the in-memory store actually reflects the
// server state. Public-mode deployments never hit this path because the
// initial fetch already succeeds.
watch(
  () => auth.authenticated,
  async (now, prev) => {
    if (now && !prev) {
      await ui.refresh()
    }
  },
)

onMounted(() => {
  auth.refresh()
  // Wallpaper / theme load runs in parallel with auth — neither blocks the
  // other. Errors inside ensureLoaded fall back to defaults silently
  // (private-mode login page would 401 here; that's expected — the watch
  // above hydrates after the user authenticates).
  ui.ensureLoaded()
})
</script>

<template>
  <NConfigProvider :theme="theme" :theme-overrides="themeOverrides">
    <!-- Wallpaper layer: fixed full-viewport, behind everything (z-index:-1).
         v0.1.7: dropped the `filter: blur()` binding. A 4K wallpaper run
         through 9 px Gaussian every frame = continuous GPU work, which
         made both home and admin feel laggy from page load (Bevan, F12
         Performance, all-red Frames at idle). Console-disabling the
         filter alone instantly returned 60 fps. The ui.blur setting
         (slider in admin/site-settings) is still persisted, just no
         longer applied — a future release can decide whether to bake
         blur into the wallpaper at upload time or hide the slider. -->
    <div
      v-if="ui.wallpaperUrl"
      class="wallpaper-layer"
      :style="{ backgroundImage: `url(${ui.wallpaperUrl})` }"
      aria-hidden="true"
    />
    <!-- v0.2.26: starfield replaces v0.2.25 aurora/glow. Canvas-driven
         (RAF + DPR + visibility pause + reduced-motion) lives inside the
         component; this layer just gates rendering on non-admin routes
         to keep the Cards drag UX paint-cost-free on /admin/*. -->
    <StarfieldBackground v-if="!isAdminRoute" />
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
  /* v0.1.7: removed `transform: translateZ(0)` + `will-change: filter`.
     They existed to GPU-accelerate the filter that's now gone; without
     a filter the layer-promotion isn't needed and the will-change hint
     would just cost memory for no benefit. The `[style*="blur"]` scale
     rule that compensated for blurred-edge bleed is also gone — no
     blur, no soft edge. */
}
</style>
