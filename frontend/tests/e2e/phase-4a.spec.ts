import { test, type Page } from '@playwright/test'
import path from 'node:path'

// Phase 4a: trusted-IP whitelist + soft-lock dashboard + drag-drop reorder
// + URL protocol prefill + CityWidget pulse loading.

const SHOTS_BASE = process.env.MOON_SHOTS_DIR ?? 'P:/moon-panel/screenshots/phase-4a'
const NOW = '2026-05-01T12:00:00+08:00'

async function mockAdmin(page: Page, totpEnabled = false) {
  await page.route('**/api/auth/me', (route) =>
    route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({
        code: 0, msg: 'ok',
        data: { initialized: true, authenticated: true, username: 'admin', totp_enabled: totpEnabled },
      }),
    }),
  )
  await page.route('**/api/admin/search-engines', (route) =>
    route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({ code: 0, msg: 'ok', data: [] }),
    }),
  )
  await page.route('**/api/admin/settings', (route) =>
    route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({ code: 0, msg: 'ok', data: { 'widget.cities': '[]', 'widget.temp_unit': 'C' } }),
    }),
  )
}

const SAMPLE_GROUPS = [
  { id: 1, name: 'Media', icon: '', sort: 10, created_at: NOW, updated_at: NOW },
  { id: 2, name: 'Tools', icon: '', sort: 20, created_at: NOW, updated_at: NOW },
]
const SAMPLE_CARDS = [
  { id: 1, group_id: 1, title: 'Jellyfin', description: '', icon: '', icon_type: 'url', url_internal: 'http://j.lan', url_external: '', url_default: 'internal', open_in_new_tab: true, sort: 10, created_at: NOW, updated_at: NOW },
  { id: 2, group_id: 1, title: 'Plex', description: '', icon: '', icon_type: 'url', url_internal: 'http://p.lan', url_external: '', url_default: 'internal', open_in_new_tab: true, sort: 20, created_at: NOW, updated_at: NOW },
  { id: 3, group_id: 2, title: 'Portainer', description: '', icon: '', icon_type: 'url', url_internal: 'http://port.lan', url_external: '', url_default: 'internal', open_in_new_tab: true, sort: 10, created_at: NOW, updated_at: NOW },
]

const SAMPLE_TRUSTED = [
  { cidr: '192.168.0.0/16', note: '家庭局域网', added_at: NOW },
  { cidr: '10.0.0.0/8', note: '公司内网', added_at: NOW },
]
const SAMPLE_LOCKED = [
  { ip: '198.51.100.22', source: 'login', failures: 5, remaining_seconds: 1693, locked_until: '2026-05-01T12:30:00Z' },
  { ip: '203.0.113.99', source: 'totp', failures: 7, remaining_seconds: 542, locked_until: '2026-05-01T12:09:00Z' },
]

function shotPath(name: string, project: string): string {
  return path.join(SHOTS_BASE, project, `${name}.png`)
}

test.describe('Phase 4a security + drag', () => {
  test('01-trusted-network-section', async ({ page }, testInfo) => {
    test.skip(testInfo.project.name === 'mobile', 'admin pages primarily desktop')
    await mockAdmin(page)
    await page.route('**/api/admin/security/trusted-ips', (route) =>
      route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({ code: 0, msg: 'ok', data: { items: SAMPLE_TRUSTED } }),
      }),
    )
    await page.goto('/admin/settings')
    await page.locator('text=受信网络（CIDR 白名单）').waitFor({ timeout: 10_000 })
    await page.evaluate(() => {
      const card = Array.from(document.querySelectorAll('.n-card')).find((c) =>
        c.textContent?.includes('受信网络'),
      )
      card?.scrollIntoView({ block: 'center' })
    })
    await page.waitForTimeout(200)
    await page.screenshot({ path: shotPath('01-trusted-network-section', testInfo.project.name), fullPage: false })
  })

  test('02-add-trusted-modal', async ({ page }, testInfo) => {
    test.skip(testInfo.project.name === 'mobile', 'admin pages primarily desktop')
    await mockAdmin(page)
    await page.route('**/api/admin/security/trusted-ips', (route) =>
      route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({ code: 0, msg: 'ok', data: { items: [] } }),
      }),
    )
    await page.goto('/admin/settings')
    await page.locator('button', { hasText: '添加 CIDR' }).waitFor({ timeout: 10_000 })
    await page.locator('button', { hasText: '添加 CIDR' }).click()
    await page.waitForSelector('.n-modal')
    await page.waitForTimeout(200)
    await page.screenshot({ path: shotPath('02-add-trusted-modal', testInfo.project.name), fullPage: false })
  })

  test('03-add-trusted-rejects-default-route', async ({ page }, testInfo) => {
    test.skip(testInfo.project.name === 'mobile', 'admin pages primarily desktop')
    await mockAdmin(page)
    await page.route('**/api/admin/security/trusted-ips', (route) => {
      if (route.request().method() === 'POST') {
        route.fulfill({
          status: 400,
          contentType: 'application/json',
          body: JSON.stringify({ code: 400, msg: 'CIDR 0.0.0.0/0 covers the entire address space; refusing to whitelist' }),
        })
      } else {
        route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify({ code: 0, msg: 'ok', data: { items: [] } }),
        })
      }
    })
    await page.goto('/admin/settings')
    await page.locator('button', { hasText: '添加 CIDR' }).waitFor({ timeout: 10_000 })
    await page.locator('button', { hasText: '添加 CIDR' }).click()
    await page.waitForSelector('.n-modal')
    await page.locator('.n-modal input').first().fill('0.0.0.0/0')
    await page.locator('.n-modal button', { hasText: '添加' }).click()
    await page.waitForSelector('.n-alert', { timeout: 3000 })
    await page.waitForTimeout(200)
    await page.screenshot({ path: shotPath('03-add-trusted-rejects-default-route', testInfo.project.name), fullPage: false })
  })

  test('04-security-page-locked-ips', async ({ page }, testInfo) => {
    test.skip(testInfo.project.name === 'mobile', 'admin pages primarily desktop')
    await mockAdmin(page)
    await page.route('**/api/admin/security/locked-ips', (route) =>
      route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({ code: 0, msg: 'ok', data: { items: SAMPLE_LOCKED } }),
      }),
    )
    await page.goto('/admin/security')
    // v0.2.17: NaiveUI internal -> own BEM (Security.vue NDataTable wrapped with class="sec__table")
    await page.waitForSelector('.sec__table tbody tr', { timeout: 10_000 })
    await page.waitForTimeout(200)
    await page.screenshot({ path: shotPath('04-security-page-locked-ips', testInfo.project.name), fullPage: false })
  })

  test('05-security-page-empty', async ({ page }, testInfo) => {
    test.skip(testInfo.project.name === 'mobile', 'admin pages primarily desktop')
    await mockAdmin(page)
    await page.route('**/api/admin/security/locked-ips', (route) =>
      route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({ code: 0, msg: 'ok', data: { items: [] } }),
      }),
    )
    await page.goto('/admin/security')
    await page.waitForSelector('.n-empty', { timeout: 10_000 })
    await page.waitForTimeout(200)
    await page.screenshot({ path: shotPath('05-security-page-empty', testInfo.project.name), fullPage: false })
  })

  // v0.2.15: 06-cards-sort-modal removed — CardsSortModal.vue + "调整顺序" button
  // deleted in commit b7fc394 (X3 inline drag fully replaces modal).
  // v0.2.16: 07-groups-sort-modal removed — GroupsSortModal.vue + "调整顺序" button
  // deleted (X3 inline drag in Groups.vue replaces modal). v0.2.17+ candidate:
  // add inline drag e2e test once <SortableTable> abstraction lands.

  test('08-card-create-with-prefill', async ({ page }, testInfo) => {
    test.skip(testInfo.project.name === 'mobile', 'admin pages primarily desktop')
    await mockAdmin(page)
    await page.route('**/api/admin/groups', (route) =>
      route.fulfill({
        status: 200, contentType: 'application/json',
        body: JSON.stringify({ code: 0, msg: 'ok', data: SAMPLE_GROUPS }),
      }),
    )
    await page.route('**/api/admin/cards', (route) =>
      route.fulfill({
        status: 200, contentType: 'application/json',
        body: JSON.stringify({ code: 0, msg: 'ok', data: [] }),
      }),
    )
    await page.goto('/admin/cards')
    await page.locator('button', { hasText: '新建卡片' }).waitFor({ timeout: 10_000 })
    await page.locator('button', { hasText: '新建卡片' }).click()
    await page.waitForSelector('.n-modal')
    await page.waitForTimeout(200)
    await page.screenshot({ path: shotPath('08-card-create-with-prefill', testInfo.project.name), fullPage: false })
  })

  test('09-citywidget-loading-pulse', async ({ page }, testInfo) => {
    // Hold the weather request so the pulse animation is visible
    await page.route('**/api/auth/me', (route) =>
      route.fulfill({
        status: 200, contentType: 'application/json',
        body: JSON.stringify({ code: 0, msg: 'ok', data: { initialized: true, authenticated: false } }),
      }),
    )
    await page.route('**/api/public/panel', (route) =>
      route.fulfill({
        status: 200, contentType: 'application/json',
        body: JSON.stringify({
          code: 0, msg: 'ok',
          data: {
            site: {
              public_mode: true, temp_unit: 'C',
              cities: [
                { name_cn: '北京', name_en: 'Beijing', tz: 'Asia/Shanghai', lat: 39.9, lon: 116.4 },
                { name_cn: '纽约', name_en: 'New York', tz: 'America/New_York', lat: 40.7, lon: -74.0 },
              ],
              network: { probe_url: '' },
            },
            groups: [], search_engines: [],
          },
        }),
      }),
    )
    // Indefinite hang on weather → pulse remains visible
    await page.route('**/api/public/weather**', () => {
      // Don't respond — test screenshot before timeout
    })
    await page.goto('/')
    await page.waitForSelector('[data-testid="home-hero"]', { timeout: 5000 })
    await page.waitForTimeout(500)
    await page.screenshot({ path: shotPath('09-citywidget-loading-pulse', testInfo.project.name), fullPage: false })
  })
})
