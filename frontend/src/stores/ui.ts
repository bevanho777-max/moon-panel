import { defineStore } from 'pinia'
import { computed, ref } from 'vue'
import type { GlobalThemeOverrides } from 'naive-ui'
import { updateSettings } from '@/api/setting'
import { getPanel } from '@/api/panel'

/**
 * UI customization state — wallpaper / blur / theme primary.
 *
 * Single source of truth for these three settings across the app:
 *   - App.vue reads `themeOverrides` and passes to NConfigProvider
 *   - WallpaperLayer reads `wallpaperUrl` + `blur` to render the background
 *   - SiteSettings.vue mutates via the action methods
 *
 * Why Pinia (vs localStorage events): admin and home are different routes
 * but share the same app instance. A reactive store lets admin changes
 * reflect in the home preview without polling or storage events. Persistence
 * still goes through the backend `/api/admin/settings` endpoint — the store
 * is just the in-memory cache that App.vue and Home.vue subscribe to.
 *
 * Initial load: ensureLoaded() pulls from /api/public/panel (which is what
 * Home loads anyway) — no extra request on cold start. Admin pages call
 * setWallpaper/setBlur/setThemePrimary, which optimistically update the
 * store and then PUT to /admin/settings.
 */
export const useUIStore = defineStore('ui', () => {
  const wallpaper = ref<string | null>(null)
  const blur = ref<number>(0)
  const themePrimary = ref<string | null>(null)
  const builtins = ref<string[]>([])
  // v0.2.0: site.title is admin-editable (defaults to "Moon Panel"). Lives
  // in the ui store because it's display chrome — same lifecycle as
  // wallpaper/theme — and gets hydrated by the same getPanel() call.
  const siteTitle = ref<string>('Moon Panel')
  // v0.2.1: theme preset. "moon" (default v0.2.0 visual) or "risen" (warm
  // serif). Watched in App.vue for body data-theme + lazy font load.
  const themePreset = ref<'moon' | 'risen'>('moon')
  const loaded = ref(false)
  const loading = ref(false)

  // The wallpaper layer reads this directly. resolveWallpaperURL is duplicated
  // here (rather than imported from api/wallpaper.ts) so the store has zero
  // dependency on the wallpaper API module — keeps the dependency graph
  // shallow for the App.vue critical path.
  const wallpaperUrl = computed<string | null>(() => {
    const v = wallpaper.value
    if (!v) return null
    if (v.startsWith('builtin:')) return `/assets/wallpapers/${v.slice('builtin:'.length)}.svg`
    if (v.startsWith('upload:')) return `/uploads/${v.slice('upload:'.length)}`
    return null
  })

  const themeOverrides = computed<GlobalThemeOverrides>(() => {
    // v0.2.22: NaiveUI Input/Select/Button surface 跟主题切换 (Bevan daily UX
    // 反馈, "搜索框/自动跟随框不管什么主题都默认"). D.3.b: cssr inject CSS var
    // 字符串, 浏览器 paint 时解析 var(), [data-theme] 改变自动 invalidate —
    // 0 reactivity hack. Focus 走 γ fallback 'var(--mp-search-bg-focus,
    // var(--mp-search-bg))' future-proof. NaiveUI typed key 确认:
    // Button=colorHover/colorPressed/colorFocus, InternalSelection=colorActive.
    // 不动 colorPrimary 字段, primary button 走下方 common.primaryColor* 联动.
    const base: GlobalThemeOverrides = {
      Input: {
        color: 'var(--mp-search-bg)',
        colorFocus: 'var(--mp-search-bg-focus, var(--mp-search-bg))',
        borderColor: 'var(--mp-search-border)',
        borderColorHover: 'var(--mp-search-border)',
        borderColorFocus: 'var(--mp-search-border)',
        textColor: 'var(--mp-text-primary)',
      },
      InternalSelection: {
        color: 'var(--mp-search-bg)',
        colorActive: 'var(--mp-search-bg-focus, var(--mp-search-bg))',
        border: '1px solid var(--mp-search-border)',
        borderHover: '1px solid var(--mp-search-border)',
        borderActive: '1px solid var(--mp-search-border)',
        borderFocus: '1px solid var(--mp-search-border)',
        textColor: 'var(--mp-text-primary)',
      },
      Button: {
        color: 'var(--mp-search-bg)',
        colorHover: 'var(--mp-search-bg-focus, var(--mp-search-bg))',
        colorPressed: 'var(--mp-search-bg-focus, var(--mp-search-bg))',
        colorFocus: 'var(--mp-search-bg-focus, var(--mp-search-bg))',
        border: '1px solid var(--mp-search-border)',
        borderHover: '1px solid var(--mp-search-border)',
        borderFocus: '1px solid var(--mp-search-border)',
        textColor: 'var(--mp-text-primary)',
      },
    }
    if (themePrimary.value) {
      const shades = derivePrimaryShades(themePrimary.value)
      base.common = {
        primaryColor: shades.base,
        primaryColorHover: shades.hover,
        primaryColorPressed: shades.pressed,
        primaryColorSuppl: shades.suppl,
      }
    }
    return base
  })

  /** Load from backend if not already loaded. Idempotent.
   *
   *  Reuses the public panel endpoint, so calling this from Home.vue is "free"
   *  (it would have fired anyway). On private-mode deployments the login page
   *  hits a 401 — we swallow the error and mark loaded=true with defaults so
   *  we don't infinite-retry. After login, the auth flow can call refresh()
   *  to re-pull. */
  async function ensureLoaded() {
    if (loaded.value || loading.value) return
    loading.value = true
    try {
      const panel = await getPanel()
      const ui = panel.site.ui
      wallpaper.value = ui.wallpaper
      blur.value = ui.wallpaper_blur
      themePrimary.value = ui.theme_primary
      builtins.value = ui.builtins
      siteTitle.value = panel.site.title || 'Moon Panel'
      themePreset.value = panel.site.theme_preset === 'risen' ? 'risen' : 'moon'
      loaded.value = true
    } catch {
      // 401 or network error: keep defaults, mark loaded so the bg layer
      // stops showing a "loading" state. Will re-pull via refresh() after
      // a successful login.
      loaded.value = true
    } finally {
      loading.value = false
    }
  }

  /** Force re-pull from server. Used after login (private mode) when the
   *  initial cold-start ensureLoaded() got 401'd and skipped the real data. */
  async function refresh() {
    loaded.value = false
    await ensureLoaded()
  }

  async function setWallpaper(spec: string | null) {
    const prev = wallpaper.value
    wallpaper.value = spec
    try {
      await updateSettings({ 'ui.wallpaper': spec ?? '' })
    } catch (e) {
      wallpaper.value = prev
      throw e
    }
  }

  async function setBlur(px: number) {
    const clamped = Math.max(0, Math.min(20, Math.round(px)))
    const prev = blur.value
    blur.value = clamped
    try {
      await updateSettings({ 'ui.wallpaper_blur': String(clamped) })
    } catch (e) {
      blur.value = prev
      throw e
    }
  }

  async function setThemePrimary(hex: string | null) {
    // Normalize: lowercase, ensure leading #. Empty string also serializes as
    // "no override" on the backend (loadUISettings treats "" as nil).
    const normalized = hex ? '#' + hex.replace(/^#/, '').toLowerCase() : null
    const prev = themePrimary.value
    themePrimary.value = normalized
    try {
      await updateSettings({ 'ui.theme_primary': normalized ?? '' })
    } catch (e) {
      themePrimary.value = prev
      throw e
    }
  }

  /** Optimistic local-only update — no backend round-trip. Used by the blur
   *  slider during drag so the preview is smooth (the real PUT fires on
   *  drag-end via setBlur). */
  function previewBlur(px: number) {
    blur.value = Math.max(0, Math.min(20, Math.round(px)))
  }

  // v0.2.3: theme → recommended wallpaper map. moon's daily-driver visual
  // pairs with the night sky; risen's warm golden palette pairs with the
  // starlit-dunes wallpaper introduced this release. Builtin-only — users
  // who picked their own upload keep it on theme switch.
  const THEME_WALLPAPER_MAP: Record<'moon' | 'risen', string> = {
    moon: 'builtin:night',
    risen: 'builtin:starlit_dunes',
  }

  /** v0.2.1+ (v0.2.3 update): theme preset setter. Persists "site.theme_preset"
   *  via the generic /admin/settings endpoint, optimistic.
   *  v0.2.3: when the user is on a builtin wallpaper, also auto-swap to the
   *  theme's recommended builtin so the panel feels visually coherent
   *  immediately after switching themes. Custom uploads are preserved
   *  (they're a personal choice; we don't override). Both updates ride a
   *  single /admin/settings PUT so they land atomically. */
  async function setThemePreset(preset: 'moon' | 'risen') {
    const prevTheme = themePreset.value
    const prevWallpaper = wallpaper.value
    themePreset.value = preset

    const payload: Record<string, string> = { 'site.theme_preset': preset }
    const recommended = THEME_WALLPAPER_MAP[preset]
    const isOnBuiltin = !!prevWallpaper && prevWallpaper.startsWith('builtin:')
    const shouldSwapWallpaper = isOnBuiltin && prevWallpaper !== recommended
    if (shouldSwapWallpaper) {
      payload['ui.wallpaper'] = recommended
      wallpaper.value = recommended
    }

    try {
      await updateSettings(payload)
    } catch (e) {
      themePreset.value = prevTheme
      if (shouldSwapWallpaper) wallpaper.value = prevWallpaper
      throw e
    }
  }

  /** v0.2.0: site title setter. Persists "site.title" via /admin/settings.
   *  Empty string clears back to backend default ("Moon Panel"). */
  async function setSiteTitle(title: string) {
    const trimmed = title.trim()
    const prev = siteTitle.value
    siteTitle.value = trimmed || 'Moon Panel'
    try {
      await updateSettings({ 'site.title': trimmed })
    } catch (e) {
      siteTitle.value = prev
      throw e
    }
  }

  /** Same idea for color picker live-preview. */
  function previewThemePrimary(hex: string | null) {
    themePrimary.value = hex ? '#' + hex.replace(/^#/, '').toLowerCase() : null
  }

  return {
    wallpaper,
    blur,
    themePrimary,
    builtins,
    siteTitle,
    themePreset,
    loaded,
    loading,
    wallpaperUrl,
    themeOverrides,
    ensureLoaded,
    refresh,
    setWallpaper,
    setBlur,
    setThemePrimary,
    setSiteTitle,
    setThemePreset,
    previewBlur,
    previewThemePrimary,
  }
})

/**
 * Derive 4 NaiveUI primary shades from a base hex. NaiveUI's default theme
 * uses base / hover / pressed / suppl that are roughly:
 *   hover   = base lightened ~8% in HSL
 *   pressed = base darkened ~8% in HSL
 *   suppl   = base lightened ~4% (used in subtle backgrounds)
 *
 * We approximate that by bumping HSL lightness. Not a perfect match for the
 * official `createPrimaryColor` function in naive-ui internals, but visually
 * close and avoids depending on a private API.
 */
function derivePrimaryShades(hex: string): {
  base: string
  hover: string
  pressed: string
  suppl: string
} {
  const base = normalizeHex(hex)
  const { h, s, l } = hexToHsl(base)
  return {
    base,
    hover: hslToHex(h, s, clampL(l + 0.08)),
    pressed: hslToHex(h, s, clampL(l - 0.08)),
    suppl: hslToHex(h, s, clampL(l + 0.04)),
  }
}

function normalizeHex(hex: string): string {
  let v = hex.trim().replace(/^#/, '').toLowerCase()
  if (v.length === 3) v = v.split('').map((c) => c + c).join('')
  return '#' + v
}

function clampL(l: number): number {
  return Math.max(0, Math.min(1, l))
}

function hexToHsl(hex: string): { h: number; s: number; l: number } {
  const v = hex.replace(/^#/, '')
  const r = parseInt(v.slice(0, 2), 16) / 255
  const g = parseInt(v.slice(2, 4), 16) / 255
  const b = parseInt(v.slice(4, 6), 16) / 255
  const max = Math.max(r, g, b)
  const min = Math.min(r, g, b)
  const l = (max + min) / 2
  let h = 0
  let s = 0
  if (max !== min) {
    const d = max - min
    s = l > 0.5 ? d / (2 - max - min) : d / (max + min)
    switch (max) {
      case r: h = ((g - b) / d + (g < b ? 6 : 0)) / 6; break
      case g: h = ((b - r) / d + 2) / 6; break
      case b: h = ((r - g) / d + 4) / 6; break
    }
  }
  return { h, s, l }
}

function hslToHex(h: number, s: number, l: number): string {
  let r: number, g: number, b: number
  if (s === 0) {
    r = g = b = l
  } else {
    const q = l < 0.5 ? l * (1 + s) : l + s - l * s
    const p = 2 * l - q
    r = hueToRgb(p, q, h + 1 / 3)
    g = hueToRgb(p, q, h)
    b = hueToRgb(p, q, h - 1 / 3)
  }
  const toHex = (n: number) => Math.round(n * 255).toString(16).padStart(2, '0')
  return '#' + toHex(r) + toHex(g) + toHex(b)
}

function hueToRgb(p: number, q: number, t: number): number {
  if (t < 0) t += 1
  if (t > 1) t -= 1
  if (t < 1 / 6) return p + (q - p) * 6 * t
  if (t < 1 / 2) return q
  if (t < 2 / 3) return p + (q - p) * (2 / 3 - t) * 6
  return p
}

/** Exported for SiteSettings preset palette. Five hand-tuned shades — chosen
 *  to be visibly distinct, work on dark backgrounds, and match common admin
 *  palette expectations (blue=neutral, purple=brand, green=success, etc). */
export const THEME_PRESETS: ReadonlyArray<{ name: string; hex: string }> = [
  { name: '蓝', hex: '#5b8def' },
  { name: '紫', hex: '#9b6bdf' },
  { name: '绿', hex: '#4caf6f' },
  { name: '粉', hex: '#e26d8a' },
  { name: '橙', hex: '#ed8936' },
] as const
