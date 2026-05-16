import { test, type Page, type Route } from '@playwright/test'
import fs from 'node:fs'
import path from 'node:path'

// Marketing screenshots for the public README. Four shots:
//   home-desktop, home-mobile, admin-cards, admin-site-settings
// Output goes to docs/screenshots/ (NOT screenshots/phase-X/) — these
// are committed README assets, not phase-tracking captures.
//
// All API calls are mocked via page.route() so no Go backend is needed.
// Wallpaper SVG is read from disk and served inline so the wallpaper layer
// renders correctly under vite preview.
//
// Paths are absolute because Playwright runs in C:/moon-build/frontend
// (F-lite copy with node_modules) but the screenshots and source assets
// live in C:/moon-panel-dev (the committed repo).

const SHOTS_DIR =
  process.env.MOON_README_SHOTS_DIR ?? 'C:/moon-panel-dev/docs/screenshots'

const AURORA_SVG_PATH =
  process.env.MOON_AURORA_SVG ??
  'C:/moon-panel-dev/backend/internal/assets/wallpapers/aurora.svg'

const NOW = '2026-05-02T12:00:00+08:00'

// 5 cards across 2 groups. Lucide icons are bundled with the frontend so
// they render without external network. Mix of internal-only / external-only
// / both shows the NetworkSwitcher distinction visually.
const SAMPLE_GROUPS = [
  {
    id: 1,
    name: 'Media',
    icon: '',
    sort: 10,
    created_at: NOW,
    updated_at: NOW,
    cards: [
      {
        id: 1,
        group_id: 1,
        title: 'Jellyfin',
        description: 'Self-hosted media server',
        icon: 'lucide:film',
        icon_type: 'lucide',
        url_internal: 'http://jellyfin.lan:8096',
        url_external: 'https://media.example.com',
        url_default: 'internal',
        open_in_new_tab: true,
        sort: 10,
        created_at: NOW,
        updated_at: NOW,
      },
      {
        id: 2,
        group_id: 1,
        title: 'Plex',
        description: 'Streaming server',
        icon: 'lucide:tv',
        icon_type: 'lucide',
        url_internal: 'http://plex.lan:32400',
        url_external: '',
        url_default: 'internal',
        open_in_new_tab: true,
        sort: 20,
        created_at: NOW,
        updated_at: NOW,
      },
      {
        id: 3,
        group_id: 1,
        title: 'Photoprism',
        description: 'Photo library',
        icon: 'lucide:image',
        icon_type: 'lucide',
        url_internal: 'http://photos.lan:2342',
        url_external: 'https://photos.example.com',
        url_default: 'internal',
        open_in_new_tab: true,
        sort: 30,
        created_at: NOW,
        updated_at: NOW,
      },
    ],
  },
  {
    id: 2,
    name: 'Tools',
    icon: '',
    sort: 20,
    created_at: NOW,
    updated_at: NOW,
    cards: [
      {
        id: 4,
        group_id: 2,
        title: 'Home Assistant',
        description: 'Home automation hub',
        icon: 'lucide:home',
        icon_type: 'lucide',
        url_internal: 'http://hass.lan:8123',
        url_external: '',
        url_default: 'internal',
        open_in_new_tab: true,
        sort: 10,
        created_at: NOW,
        updated_at: NOW,
      },
      {
        id: 5,
        group_id: 2,
        title: 'Pi-hole',
        description: 'Network-wide ad blocker',
        icon: 'lucide:shield',
        icon_type: 'lucide',
        url_internal: 'http://pihole.lan/admin',
        url_external: '',
        url_default: 'internal',
        open_in_new_tab: true,
        sort: 20,
        created_at: NOW,
        updated_at: NOW,
      },
    ],
  },
]

const SAMPLE_ENGINES = [
  {
    id: 1,
    name: 'Google',
    url_template: 'https://www.google.com/search?q={query}',
    icon: '',
    is_default: true,
    sort: 10,
  },
  {
    id: 2,
    name: 'DuckDuckGo',
    url_template: 'https://duckduckgo.com/?q={query}',
    icon: '',
    is_default: false,
    sort: 20,
  },
]

const SAMPLE_CITIES = [
  { name: 'New York', lat: 40.71, lon: -74.01, country: 'US' },
  { name: 'Tokyo', lat: 35.68, lon: 139.65, country: 'JP' },
]

const SAMPLE_UI = {
  wallpaper: 'builtin:aurora',
  wallpaper_blur: 8,
  theme_primary: '#9b6bdf',
  builtins: ['aurora', 'graphite', 'night'],
}

function jsonFulfill(route: Route, body: unknown) {
  return route.fulfill({
    status: 200,
    contentType: 'application/json',
    body: JSON.stringify(body),
  })
}

// Read aurora SVG once at import time. If the file isn't reachable, fall
// back to a solid color — the test still passes, screenshot just looks
// less polished. Caller should ensure the path is right; failing loud
// here would mask a more interesting failure downstream.
let auroraSvg = ''
try {
  auroraSvg = fs.readFileSync(AURORA_SVG_PATH, 'utf8')
} catch {
  auroraSvg = '<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 1 1"><rect width="1" height="1" fill="#1d2c5c"/></svg>'
}

async function mockCommon(page: Page, opts: { authenticated: boolean }) {
  // v0.2.23 C.2: pin session network to 'lan' before the Home page loads.
  // Without this, README screenshots would capture the WAN-disabled state of
  // lan-only cards (Plex / Hass / Pi-hole) because headless fetches to
  // *.lan addresses always fail. Key mirrors KEY_SESSION_OVERRIDE in
  // src/stores/network.ts.
  await page.addInitScript(() => {
    try {
      sessionStorage.setItem('moon-panel.session-override', 'lan')
    } catch {
      /* sessionStorage quota errors aren't relevant in playwright */
    }
  })

  // Public-mode auth state. The home page renders without auth; admin
  // routes need authenticated:true.
  await page.route('**/api/auth/me', (route) =>
    jsonFulfill(route, {
      code: 0,
      msg: 'ok',
      data: {
        initialized: true,
        authenticated: opts.authenticated,
        username: opts.authenticated ? 'admin' : '',
        totp_enabled: false,
      },
    }),
  )

  // Public panel — drives Home + the UI store wallpaper/theme.
  await page.route('**/api/public/panel', (route) =>
    jsonFulfill(route, {
      code: 0,
      msg: 'ok',
      data: {
        site: {
          public_mode: true,
          cities: SAMPLE_CITIES,
          temp_unit: 'C',
          ui: SAMPLE_UI,
          network: { probe_url: '' },
        },
        groups: SAMPLE_GROUPS,
        search_engines: SAMPLE_ENGINES,
      },
    }),
  )

  // Wallpaper SVG: serve the real builtin file inline so the gradient
  // renders correctly under vite preview (which has no Go backend).
  await page.route('**/assets/wallpapers/aurora.svg', (route) =>
    route.fulfill({
      status: 200,
      contentType: 'image/svg+xml',
      body: auroraSvg,
    }),
  )

  // HomeHero weather. Each city fires one request. Stable values for
  // reproducible screenshots (sunny daytime, ~18°C / ~25°C).
  await page.route('**/api/public/weather*', (route) => {
    const url = new URL(route.request().url())
    const lat = parseFloat(url.searchParams.get('lat') ?? '0')
    const isWarm = lat < 36
    return jsonFulfill(route, {
      latitude: lat,
      longitude: parseFloat(url.searchParams.get('lon') ?? '0'),
      timezone: 'UTC',
      current: {
        time: NOW,
        temperature_2m: isWarm ? 25 : 18,
        weather_code: 0,
        is_day: 1,
      },
    })
  })
}

async function mockAdmin(page: Page) {
  await mockCommon(page, { authenticated: true })

  await page.route('**/api/admin/groups', (route) =>
    jsonFulfill(route, {
      code: 0,
      msg: 'ok',
      data: SAMPLE_GROUPS.map(({ cards: _cards, ...g }) => g),
    }),
  )
  await page.route('**/api/admin/cards', (route) =>
    jsonFulfill(route, {
      code: 0,
      msg: 'ok',
      data: SAMPLE_GROUPS.flatMap((g) => g.cards),
    }),
  )
  await page.route('**/api/admin/search-engines', (route) =>
    jsonFulfill(route, { code: 0, msg: 'ok', data: SAMPLE_ENGINES }),
  )
  await page.route('**/api/admin/settings', (route) =>
    jsonFulfill(route, {
      code: 0,
      msg: 'ok',
      data: {
        'widget.cities': JSON.stringify(SAMPLE_CITIES),
        'widget.temp_unit': 'C',
        'ui.wallpaper': SAMPLE_UI.wallpaper,
        'ui.wallpaper_blur': String(SAMPLE_UI.wallpaper_blur),
        'ui.theme_primary': SAMPLE_UI.theme_primary,
      },
    }),
  )
  await page.route('**/api/admin/security/trusted-ips', (route) =>
    jsonFulfill(route, { code: 0, msg: 'ok', data: { items: [] } }),
  )
  await page.route('**/api/admin/wallpapers', (route) =>
    jsonFulfill(route, { code: 0, msg: 'ok', data: [] }),
  )
}

function shotPath(name: string): string {
  return path.join(SHOTS_DIR, `${name}.png`)
}

test.describe('README marketing screenshots', () => {
  // These screenshots are committed deliverables under docs/screenshots/,
  // not regression assets. They're regenerated manually when the UI
  // changes; CI doesn't need to redo them, and the default output path
  // is Windows-only anyway. Skip the whole describe block on CI runners.
  test.skip(!!process.env.CI, 'README screenshots are committed manually')

  test('home-desktop', async ({ page }, info) => {
    test.skip(info.project.name === 'mobile', 'desktop-only viewport')
    await mockCommon(page, { authenticated: false })
    await page.goto('/')
    // Wait for at least one card to render — the panel API is mocked, so
    // this should resolve in <1s once the network idle settles.
    await page.locator('text=Jellyfin').first().waitFor({ timeout: 10_000 })
    // Give the wallpaper layer + acrylic backdrop-filter a moment to paint.
    await page.waitForTimeout(400)
    await page.screenshot({ path: shotPath('home-desktop'), fullPage: false })
  })

  test('home-mobile', async ({ page }, info) => {
    test.skip(info.project.name === 'desktop', 'mobile-only viewport')
    await mockCommon(page, { authenticated: false })
    await page.goto('/')
    await page.locator('text=Jellyfin').first().waitFor({ timeout: 10_000 })
    await page.waitForTimeout(400)
    await page.screenshot({ path: shotPath('home-mobile'), fullPage: false })
  })

  test('admin-cards', async ({ page }, info) => {
    test.skip(info.project.name === 'mobile', 'admin pages are desktop-first')
    await mockAdmin(page)
    await page.goto('/admin/cards')
    await page.locator('text=Jellyfin').first().waitFor({ timeout: 10_000 })
    await page.waitForTimeout(400)
    await page.screenshot({ path: shotPath('admin-cards'), fullPage: false })
  })

  test('admin-site-settings', async ({ page }, info) => {
    test.skip(info.project.name === 'mobile', 'admin pages are desktop-first')
    await mockAdmin(page)
    await page.goto('/admin/settings')
    // The settings page is long; we want the wallpaper + theme + blur block.
    // Wait for the wallpaper card to render, then scroll it into view so
    // both wallpaper and theme color cards are in the captured viewport.
    await page.locator('text=背景壁纸').first().waitFor({ timeout: 10_000 })
    await page.evaluate(() => {
      const el = Array.from(document.querySelectorAll('.n-card')).find(
        (n) => n.textContent?.includes('背景壁纸'),
      )
      el?.scrollIntoView({ block: 'start', behavior: 'instant' as ScrollBehavior })
    })
    await page.waitForTimeout(400)
    await page.screenshot({ path: shotPath('admin-site-settings'), fullPage: false })
  })
})
