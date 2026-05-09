import { test, type Page } from '@playwright/test'
import path from 'node:path'

// Always write screenshots to the source-of-truth on P: drive (NAS SMB share),
// not the F-lite C: build copy. Override via MOON_SHOTS_DIR for CI/non-Windows.
const SHOTS_BASE = process.env.MOON_SHOTS_DIR ?? 'P:/moon-panel/screenshots/phase-3c-2'

const ENGINES_MOCK = [
  { id: 1, name: 'Google', url_template: 'https://www.google.com/search?q={query}', icon: 'https://cdn.jsdelivr.net/gh/walkxcode/dashboard-icons/png/google.png', is_default: true, sort: 10, created_at: '', updated_at: '' },
  { id: 2, name: 'Bing', url_template: 'https://www.bing.com/search?q={query}', icon: 'https://cdn.jsdelivr.net/gh/walkxcode/dashboard-icons/png/bing.png', is_default: false, sort: 20, created_at: '', updated_at: '' },
  { id: 3, name: 'DuckDuckGo', url_template: 'https://duckduckgo.com/?q={query}', icon: 'https://cdn.jsdelivr.net/gh/walkxcode/dashboard-icons/png/duckduckgo.png', is_default: false, sort: 30, created_at: '', updated_at: '' },
  { id: 4, name: '百度', url_template: 'https://www.baidu.com/s?wd={query}', icon: 'https://cdn.jsdelivr.net/gh/walkxcode/dashboard-icons/png/baidu.png', is_default: false, sort: 40, created_at: '', updated_at: '' },
]

// walkxcode/dashboard-icons CDN — same source Phase 3a will integrate as the
// canonical icon library, so the mock visuals match the future production look.
const WX = 'https://cdn.jsdelivr.net/gh/walkxcode/dashboard-icons/png'

function mockCard(
  id: number,
  groupId: number,
  title: string,
  icon: string,
  description: string,
  urls: { url_internal?: string; url_external?: string; url_default?: 'internal' | 'external' } = {},
) {
  return {
    id,
    group_id: groupId,
    title,
    description,
    icon,
    icon_type: 'url',
    url_internal: urls.url_internal ?? '',
    url_external: urls.url_external ?? '',
    url_default: urls.url_default ?? 'internal',
    open_in_new_tab: true,
    sort: id * 10,
    created_at: NOW,
    updated_at: NOW,
  }
}

const NOW = '2026-04-28T12:00:00+08:00'

const HERO_CITIES = [
  { name_cn: '北京', name_en: 'Beijing', tz: 'Asia/Shanghai', lat: 39.9042, lon: 116.4074 },
  { name_cn: '纽约', name_en: 'New York', tz: 'America/New_York', lat: 40.7128, lon: -74.006 },
  { name_cn: '东京', name_en: 'Tokyo', tz: 'Asia/Tokyo', lat: 35.6762, lon: 139.6503 },
]

// Stub Open-Meteo response. weather_code 1 → 多云, is_day=1 → ⛅
function weatherStub(temperature_2m: number, weather_code: number, is_day: 0 | 1) {
  return {
    latitude: 0, longitude: 0, timezone: 'auto',
    current: {
      time: '2026-04-28T12:00',
      interval: 600,
      temperature_2m, weather_code, is_day,
    },
  }
}

const SAMPLE_PANEL = {
  code: 0,
  msg: 'ok',
  data: {
    site: { public_mode: true, cities: HERO_CITIES, temp_unit: 'C' },
    groups: [
      {
        id: 1,
        name: 'Media',
        icon: '',
        sort: 10,
        created_at: NOW,
        updated_at: NOW,
        cards: [
          mockCard(1, 1, 'Jellyfin', `${WX}/jellyfin.png`, '本地影音服务器', { url_internal: 'http://192.168.1.10:8096', url_external: 'https://media.example.com' }),
          mockCard(2, 1, 'Plex', `${WX}/plex.png`, '', { url_internal: '', url_external: 'https://plex.example.com', url_default: 'external' }),
          mockCard(3, 1, 'Emby', `${WX}/emby.png`, '', { url_internal: 'http://192.168.1.10:8920' }),
          mockCard(4, 1, 'Navidrome', `${WX}/navidrome.png`, '音乐流媒体', { url_internal: 'http://192.168.1.10:4533' }),
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
          mockCard(5, 2, 'Portainer', `${WX}/portainer.png`, 'Docker 管理', { url_internal: 'http://192.168.1.10:9443', url_external: '' }),
          mockCard(6, 2, 'Nextcloud', `${WX}/nextcloud.png`, '私有云盘', { url_internal: 'http://192.168.1.10:8080', url_external: 'https://cloud.example.com' }),
          mockCard(7, 2, 'Vaultwarden', `${WX}/vaultwarden.png`, '密码管理', { url_external: 'https://vault.example.com', url_default: 'external' }),
          mockCard(8, 2, 'Home Assistant', `${WX}/home-assistant.png`, '智能家居', { url_internal: 'http://192.168.1.10:8123' }),
        ],
      },
    ],
    search_engines: ENGINES_MOCK,
  },
}

const EMPTY_PANEL = {
  code: 0,
  msg: 'ok',
  data: {
    site: { public_mode: true, cities: [], temp_unit: 'C' },
    groups: [],
    search_engines: [],
  },
}

const ME_NOT_AUTHED = {
  code: 0,
  msg: 'ok',
  data: { initialized: true, authenticated: false },
}

async function mockMe(page: Page) {
  await page.route('**/api/auth/me', (route) =>
    route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify(ME_NOT_AUTHED),
    }),
  )
}

async function mockPanel(page: Page, body: unknown, status = 200) {
  await page.route('**/api/public/panel', (route) =>
    route.fulfill({
      status,
      contentType: 'application/json',
      body: JSON.stringify(body),
    }),
  )
  // HomeHero fires GET /api/public/weather on mount; auto-stub so test
  // screenshots don't depend on real network. Tests that want to assert
  // the loading-state can mock weather separately before this.
  await mockWeatherDefault(page)
}

// Default weather mock: deterministic temperature per city based on lat (so the
// 3 stub widgets show distinct values). All cities → ⛅ (weather_code=1, day).
async function mockWeatherDefault(page: Page) {
  await page.route('**/api/public/weather**', (route) => {
    const u = new URL(route.request().url())
    const lat = parseFloat(u.searchParams.get('lat') ?? '0')
    // Map Beijing(~40)→18, NY(~40 negative-lon)→14, Tokyo(~35)→22 — close enough
    // for distinct visual; doesn't need to match real climates.
    const lon = parseFloat(u.searchParams.get('lon') ?? '0')
    let temp = 20
    if (lat > 39 && lon > 100) temp = 18 // Beijing
    else if (lat > 39 && lon < 0) temp = 14 // NY
    else if (lat > 30 && lon > 130) temp = 22 // Tokyo
    route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify(weatherStub(temp, 1, 1)),
    })
  })
}

function shotPath(name: string, project: string): string {
  return path.join(SHOTS_BASE, project, `${name}.png`)
}

// Wait until every <img> in the DOM has finished loading (success OR error).
// Without this, screenshots are racy: <img src="https://..."> can still be in
// flight when the home grid is visible, producing inconsistent captures.
async function waitForImagesSettled(page: Page) {
  await page.waitForLoadState('networkidle')
  await page.waitForFunction(() => {
    const imgs = Array.from(document.querySelectorAll('img'))
    return imgs.every((img) => img.complete)
  }, { timeout: 10_000 })
}

// LucideIcon renders async (loads lucide-vue-next chunk on first use). It
// marks itself with [data-lucide-loading=true] while the chunk is in flight.
// Wait for all such markers to disappear before screenshotting.
async function waitForLucideSettled(page: Page) {
  await page.waitForFunction(
    () => document.querySelectorAll('[data-lucide-loading]').length === 0,
    { timeout: 10_000 },
  )
}

async function waitForVisualReady(page: Page) {
  await waitForImagesSettled(page)
  await waitForLucideSettled(page)
}

// Wait for HomeHero to be in the DOM and at least one widget to have its
// weather row rendered (temp != "—"). Without this, screenshots may be taken
// while the hero is still empty/placeholder.
async function waitForHeroSettled(page: Page) {
  await page.waitForSelector('[data-testid="home-hero"]', { timeout: 5_000 })
  await page.waitForFunction(() => {
    const temps = Array.from(document.querySelectorAll('[data-testid="home-hero"] .cw__temp'))
    return temps.length > 0 && temps.every((el) => (el.textContent ?? '').trim() !== '—')
  }, { timeout: 5_000 }).catch(() => {
    // If weather still loading, screenshots will show placeholder — not fatal.
  })
}

test.describe('Phase 3c-2 home hero + time display', () => {
  test('01-home-with-cards', async ({ page }, testInfo) => {
    await mockMe(page)
    await mockPanel(page, SAMPLE_PANEL)
    await page.goto('/')
    await page.waitForSelector('.home-group')
    await waitForImagesSettled(page)
    await page.screenshot({ path: shotPath('01-home-with-cards', testInfo.project.name), fullPage: true })
  })

  test('02-home-empty-state', async ({ page }, testInfo) => {
    await mockMe(page)
    await mockPanel(page, EMPTY_PANEL)
    await page.goto('/')
    await page.waitForSelector('.home-empty')
    await waitForImagesSettled(page)
    await page.screenshot({ path: shotPath('02-home-empty-state', testInfo.project.name), fullPage: true })
  })

  test('03-network-switcher-open', async ({ page }, testInfo) => {
    await mockMe(page)
    await mockPanel(page, SAMPLE_PANEL)
    await page.goto('/')
    await page.waitForSelector('.home-group')
    await waitForImagesSettled(page)
    // Click the visible switcher trigger (desktop: NSelect; mobile: icon button)
    const trigger = testInfo.project.name === 'mobile'
      ? page.locator('[data-testid="network-switcher-narrow"]')
      : page.locator('[data-testid="network-switcher-wide"] .n-base-selection')
    await trigger.click()
    await page.waitForSelector(testInfo.project.name === 'mobile' ? '.n-dropdown-option' : '.n-base-select-option')
    await page.screenshot({ path: shotPath('03-network-switcher-open', testInfo.project.name), fullPage: true })
  })

  test('04-card-context-menu', async ({ page }, testInfo) => {
    test.skip(testInfo.project.name === 'mobile', 'right-click not applicable on mobile viewport')
    await mockMe(page)
    await mockPanel(page, SAMPLE_PANEL)
    await page.goto('/')
    await page.waitForSelector('.card-item')
    await page.locator('.card-item').first().click({ button: 'right' })
    await page.waitForSelector('.n-dropdown-option')
    await page.screenshot({ path: shotPath('04-card-context-menu', testInfo.project.name), fullPage: true })
  })

  test('05-public-mode-false-redirect-to-login', async ({ page }, testInfo) => {
    await mockMe(page)
    await mockPanel(page, { code: 401, msg: 'unauthorized' }, 401)
    await page.goto('/')
    // Wait for redirect to /login
    await page.waitForURL(/\/login/)
    await page.waitForSelector('.login__card', { timeout: 5000 }).catch(() => {})
    await page.screenshot({ path: shotPath('05-public-mode-false-redirect-to-login', testInfo.project.name), fullPage: true })
  })

  test('06-card-hover-state', async ({ page }, testInfo) => {
    test.skip(testInfo.project.name === 'mobile', 'no hover on mobile')
    await mockMe(page)
    await mockPanel(page, SAMPLE_PANEL)
    await page.goto('/')
    await page.waitForSelector('.card-item')
    await waitForImagesSettled(page)
    // Hover the second card (Plex), pause for transition (200ms) before shot
    await page.locator('.card-item').nth(1).hover()
    await page.waitForTimeout(350)
    await page.screenshot({ path: shotPath('06-card-hover-state', testInfo.project.name), fullPage: false })
  })

  test('07-lucide-icon-render', async ({ page }, testInfo) => {
    // Dedicated mock with lucide: prefix cards. Verifies the async-loaded
    // lucide-vue-next chunk renders real icons in CardItem.
    await mockMe(page)
    await mockPanel(page, {
      code: 0,
      msg: 'ok',
      data: {
        site: { public_mode: true },
        groups: [
          {
            id: 1,
            name: 'Lucide 渲染验证',
            icon: '',
            sort: 10,
            created_at: NOW,
            updated_at: NOW,
            cards: [
              mockCard(101, 1, 'Wrench', 'lucide:wrench', '工具', { url_internal: 'http://x.test' }),
              mockCard(102, 1, 'Server', 'lucide:server', '服务器', { url_internal: 'http://x.test' }),
              mockCard(103, 1, 'Shield Check', 'lucide:shield-check', '复合名（kebab→Pascal）', { url_internal: 'http://x.test' }),
              mockCard(104, 1, 'Music', 'lucide:music', '音乐', { url_internal: 'http://x.test' }),
              mockCard(105, 1, 'Database', 'lucide:database', '数据库', { url_internal: 'http://x.test' }),
              mockCard(106, 1, 'Missing', 'lucide:does-not-exist-xyz', '不存在的名（fallback "?"）', { url_internal: 'http://x.test' }),
            ],
          },
        ],
        search_engines: ENGINES_MOCK,
      },
    })
    await page.goto('/')
    await page.waitForSelector('.home-group')
    await waitForVisualReady(page)
    await page.screenshot({ path: shotPath('07-lucide-icon-render', testInfo.project.name), fullPage: true })
  })

  test('08-admin-icon-search-dropdown', async ({ page }, testInfo) => {
    test.skip(testInfo.project.name === 'mobile', 'admin editor primarily desktop UX')

    // Mock auth as logged-in admin so /admin/cards renders without login flow.
    await page.route('**/api/auth/me', (route) =>
      route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          code: 0, msg: 'ok',
          data: { initialized: true, authenticated: true, username: 'admin' },
        }),
      }),
    )
    // Need at least one group so the "新建卡片" button is enabled.
    await page.route('**/api/admin/groups', (route) =>
      route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          code: 0, msg: 'ok',
          data: [{ id: 1, name: 'Media', icon: '', sort: 10, created_at: NOW, updated_at: NOW }],
        }),
      }),
    )
    await page.route('**/api/admin/cards', (route) =>
      route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({ code: 0, msg: 'ok', data: [] }),
      }),
    )

    await page.goto('/admin/cards')
    await page.waitForSelector('.n-card', { timeout: 10_000 })
    // Open new-card modal
    await page.getByText('新建卡片').first().click()
    await page.waitForSelector('.n-modal')
    // Wait for the icon NAutoComplete to mount
    const iconInput = page.locator('.n-base-selection input, input[placeholder*="搜索图标"]').first()
    await iconInput.fill('jelly')
    // Wait for the catalog async chunk + the dropdown to render
    await page.waitForSelector('.n-auto-complete-menu', { timeout: 10_000 })
    // Wait briefly for thumbnail images to start loading
    await page.waitForTimeout(800)
    await page.screenshot({ path: shotPath('08-admin-icon-search-dropdown', testInfo.project.name), fullPage: false })
  })

  test('09-admin-site-settings-list', async ({ page }, testInfo) => {
    test.skip(testInfo.project.name === 'mobile', 'admin pages primarily desktop')
    await page.route('**/api/auth/me', (route) =>
      route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          code: 0, msg: 'ok',
          data: { initialized: true, authenticated: true, username: 'admin' },
        }),
      }),
    )
    await page.route('**/api/admin/search-engines', (route) =>
      route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          code: 0, msg: 'ok',
          data: [
            { id: 1, name: 'Google', url_template: 'https://www.google.com/search?q={query}', icon: 'https://www.google.com/favicon.ico', is_default: true, sort: 10, created_at: NOW, updated_at: NOW },
            { id: 2, name: 'Bing', url_template: 'https://www.bing.com/search?q={query}', icon: 'https://www.bing.com/favicon.ico', is_default: false, sort: 20, created_at: NOW, updated_at: NOW },
            { id: 3, name: 'DuckDuckGo', url_template: 'https://duckduckgo.com/?q={query}', icon: 'https://duckduckgo.com/favicon.ico', is_default: false, sort: 30, created_at: NOW, updated_at: NOW },
            { id: 4, name: '百度', url_template: 'https://www.baidu.com/s?wd={query}', icon: 'https://www.baidu.com/favicon.ico', is_default: false, sort: 40, created_at: NOW, updated_at: NOW },
          ],
        }),
      }),
    )

    await page.goto('/admin/settings')
    // v0.2.18: SortableTable abstraction, .engines-list__item -> .sortable-table__item
    await page.waitForSelector('.sortable-table__item', { timeout: 10_000 })
    await waitForImagesSettled(page)
    await page.screenshot({ path: shotPath('09-admin-site-settings-list', testInfo.project.name), fullPage: true })
  })

  test('10-admin-site-settings-edit-modal', async ({ page }, testInfo) => {
    test.skip(testInfo.project.name === 'mobile', 'admin pages primarily desktop')
    await page.route('**/api/auth/me', (route) =>
      route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          code: 0, msg: 'ok',
          data: { initialized: true, authenticated: true, username: 'admin' },
        }),
      }),
    )
    await page.route('**/api/admin/search-engines', (route) =>
      route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          code: 0, msg: 'ok',
          data: [
            { id: 1, name: 'Google', url_template: 'https://www.google.com/search?q={query}', icon: 'https://www.google.com/favicon.ico', is_default: true, sort: 10, created_at: NOW, updated_at: NOW },
          ],
        }),
      }),
    )

    await page.goto('/admin/settings')
    // v0.2.18: SortableTable abstraction, .engines-list__item -> .sortable-table__item
    await page.waitForSelector('.sortable-table__item', { timeout: 10_000 })
    await page.getByText('新建引擎').click()
    await page.waitForSelector('.n-modal')
    await page.waitForTimeout(300)
    await page.screenshot({ path: shotPath('10-admin-site-settings-edit-modal', testInfo.project.name), fullPage: false })
  })

  test('11-home-search-engine-dropdown', async ({ page }, testInfo) => {
    await mockMe(page)
    await mockPanel(page, SAMPLE_PANEL)
    await page.goto('/')
    await page.waitForSelector('.home-group')
    await waitForVisualReady(page)
    // Click engine icon button to open dropdown
    await page.locator('.header-search__trigger').click()
    await page.waitForSelector('.n-dropdown-option', { timeout: 5000 })
    await page.waitForTimeout(300)
    await page.screenshot({ path: shotPath('11-home-search-engine-dropdown', testInfo.project.name), fullPage: false })
  })

  test('12-home-search-filter-active', async ({ page }, testInfo) => {
    await mockMe(page)
    await mockPanel(page, SAMPLE_PANEL)
    await page.goto('/')
    await page.waitForSelector('.home-group')
    await waitForVisualReady(page)
    // Type a query that filters cards down
    const searchInput = page.locator('.header-search__input input').first()
    await searchInput.fill('plex')
    // Wait for fade transition (200ms transition + buffer)
    await page.waitForTimeout(400)
    await page.screenshot({ path: shotPath('12-home-search-filter-active', testInfo.project.name), fullPage: true })
  })

  test('13-home-search-no-match', async ({ page }, testInfo) => {
    test.skip(testInfo.project.name === 'mobile', 'duplicates 12 visually')
    await mockMe(page)
    await mockPanel(page, SAMPLE_PANEL)
    await page.goto('/')
    await page.waitForSelector('.home-group')
    await waitForVisualReady(page)
    const searchInput = page.locator('.header-search__input input').first()
    await searchInput.fill('xxxyyyzzz-no-match')
    await page.waitForTimeout(400)
    await page.screenshot({ path: shotPath('13-home-search-no-match', testInfo.project.name), fullPage: false })
  })

  test('14-admin-settings-time-display-section', async ({ page }, testInfo) => {
    test.skip(testInfo.project.name === 'mobile', 'admin pages primarily desktop')
    await page.route('**/api/auth/me', (route) =>
      route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          code: 0, msg: 'ok',
          data: { initialized: true, authenticated: true, username: 'admin' },
        }),
      }),
    )
    await page.route('**/api/admin/search-engines', (route) =>
      route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({ code: 0, msg: 'ok', data: ENGINES_MOCK }),
      }),
    )
    await page.route('**/api/admin/settings', (route) =>
      route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          code: 0, msg: 'ok',
          data: {
            'widget.cities': JSON.stringify([
              { name_cn: '北京', name_en: 'Beijing', tz: 'Asia/Shanghai', lat: 39.9042, lon: 116.4074 },
              { name_cn: '纽约', name_en: 'New York', tz: 'America/New_York', lat: 40.7128, lon: -74.006 },
              { name_cn: '东京', name_en: 'Tokyo', tz: 'Asia/Tokyo', lat: 35.6762, lon: 139.6503 },
            ]),
            'widget.temp_unit': 'C',
          },
        }),
      }),
    )

    await page.goto('/admin/settings')
    await page.waitForSelector('.ws__cities', { timeout: 10_000 })
    await waitForImagesSettled(page)
    await page.screenshot({ path: shotPath('14-admin-settings-time-display-section', testInfo.project.name), fullPage: true })
  })

  test('15-admin-city-picker-dropdown', async ({ page }, testInfo) => {
    test.skip(testInfo.project.name === 'mobile', 'admin pages primarily desktop')
    await page.route('**/api/auth/me', (route) =>
      route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          code: 0, msg: 'ok',
          data: { initialized: true, authenticated: true, username: 'admin' },
        }),
      }),
    )
    await page.route('**/api/admin/search-engines', (route) =>
      route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({ code: 0, msg: 'ok', data: ENGINES_MOCK }),
      }),
    )
    await page.route('**/api/admin/settings', (route) =>
      route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({ code: 0, msg: 'ok', data: { 'widget.cities': '[]', 'widget.temp_unit': 'C' } }),
      }),
    )

    await page.goto('/admin/settings')
    await page.waitForSelector('.ws__empty', { timeout: 10_000 })
    // Click "添加城市" button (label includes count 0/5)
    await page.locator('button').filter({ hasText: '添加城市' }).click()
    await page.waitForSelector('.n-modal')
    // Wait for catalog async chunk to load
    await page.waitForTimeout(500)
    const searchInput = page.locator('.n-modal input').first()
    await searchInput.fill('shang')
    await page.waitForTimeout(300)
    await page.screenshot({ path: shotPath('15-admin-city-picker-dropdown', testInfo.project.name), fullPage: false })
  })

  test('16-home-hero-loaded', async ({ page }, testInfo) => {
    await mockMe(page)
    await mockPanel(page, SAMPLE_PANEL)
    await page.goto('/')
    await page.waitForSelector('.home-group')
    await waitForHeroSettled(page)
    await waitForImagesSettled(page)
    await page.screenshot({ path: shotPath('16-home-hero-loaded', testInfo.project.name), fullPage: false })
  })

  test('17-home-hero-temp-fahrenheit', async ({ page }, testInfo) => {
    test.skip(testInfo.project.name === 'mobile', 'unit toggle visually identical at this size')
    await mockMe(page)
    await mockPanel(page, {
      code: 0,
      msg: 'ok',
      data: {
        ...SAMPLE_PANEL.data,
        site: { ...SAMPLE_PANEL.data.site, temp_unit: 'F' },
      },
    })
    await page.goto('/')
    await page.waitForSelector('.home-group')
    await waitForHeroSettled(page)
    await waitForImagesSettled(page)
    await page.screenshot({ path: shotPath('17-home-hero-temp-fahrenheit', testInfo.project.name), fullPage: false })
  })

  test('18-home-hero-five-cities', async ({ page }, testInfo) => {
    // Stress: max-out cities array (5). On mobile this should wrap to 2-col / 1-col.
    const FIVE = [
      ...HERO_CITIES,
      { name_cn: '伦敦', name_en: 'London', tz: 'Europe/London', lat: 51.5074, lon: -0.1278 },
      { name_cn: '悉尼', name_en: 'Sydney', tz: 'Australia/Sydney', lat: -33.8688, lon: 151.2093 },
    ]
    await mockMe(page)
    await mockPanel(page, {
      code: 0,
      msg: 'ok',
      data: {
        ...SAMPLE_PANEL.data,
        site: { ...SAMPLE_PANEL.data.site, cities: FIVE },
      },
    })
    await page.goto('/')
    await page.waitForSelector('.home-group')
    await waitForHeroSettled(page)
    await waitForImagesSettled(page)
    await page.screenshot({ path: shotPath('18-home-hero-five-cities', testInfo.project.name), fullPage: false })
  })

  test('19-home-hero-empty-cities', async ({ page }, testInfo) => {
    test.skip(testInfo.project.name === 'mobile', 'identical to desktop — no hero rendered')
    // When admin has cleared all cities, hero band should not render at all.
    await mockMe(page)
    await mockPanel(page, {
      code: 0,
      msg: 'ok',
      data: {
        ...SAMPLE_PANEL.data,
        site: { ...SAMPLE_PANEL.data.site, cities: [] },
      },
    })
    await page.goto('/')
    await page.waitForSelector('.home-group')
    await waitForImagesSettled(page)
    await page.screenshot({ path: shotPath('19-home-hero-empty-cities', testInfo.project.name), fullPage: false })
  })
})
