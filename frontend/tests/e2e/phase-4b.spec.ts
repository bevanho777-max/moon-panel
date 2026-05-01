import { test, type Page } from '@playwright/test'
import path from 'node:path'

// Phase 4b: StatefulInput 4-state machine + group rename stale fix.
// All scenarios use the SearchEngine editor (admin/settings) since it has
// 3 StatefulInput fields with non-empty originals — easiest reproducible
// case for state-machine assertions.

const SHOTS_BASE = process.env.MOON_SHOTS_DIR ?? 'P:/moon-panel/screenshots/phase-4b'
const NOW = '2026-05-01T12:00:00+08:00'

const GOOGLE_ENGINE = {
  id: 1,
  name: 'Google',
  url_template: 'https://www.google.com/search?q={query}',
  icon: 'https://cdn.jsdelivr.net/gh/walkxcode/dashboard-icons/png/google.png',
  is_default: true,
  sort: 10,
  created_at: NOW,
  updated_at: NOW,
}

async function mockAdmin(page: Page) {
  await page.route('**/api/auth/me', (route) =>
    route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({
        code: 0, msg: 'ok',
        data: { initialized: true, authenticated: true, username: 'admin', totp_enabled: false },
      }),
    }),
  )
  await page.route('**/api/admin/search-engines', (route) =>
    route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({ code: 0, msg: 'ok', data: [GOOGLE_ENGINE] }),
    }),
  )
  await page.route('**/api/admin/settings', (route) =>
    route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({ code: 0, msg: 'ok', data: { 'widget.cities': '[]', 'widget.temp_unit': 'C' } }),
    }),
  )
  await page.route('**/api/admin/security/trusted-ips', (route) =>
    route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({ code: 0, msg: 'ok', data: { items: [] } }),
    }),
  )
}

async function openEngineEditor(page: Page) {
  await page.goto('/admin/settings')
  await page.waitForSelector('.n-data-table-tr', { timeout: 10_000 })
  // Click "编辑" on the Google row
  await page.locator('.n-data-table-tr').filter({ hasText: 'Google' }).locator('button:has-text("编辑")').click()
  await page.waitForSelector('.n-modal')
  // Wait for StatefulInput to mount
  await page.waitForSelector('.si', { timeout: 5000 })
}

function shotPath(name: string, project: string): string {
  return path.join(SHOTS_BASE, project, `${name}.png`)
}

async function nameInput(page: Page) {
  // First StatefulInput in the engine editor is the "名称" field.
  return page.locator('.si').first().locator('input')
}

test.describe('Phase 4b StatefulInput state machine', () => {
  test('01-state-A-idle-shows-original', async ({ page }, testInfo) => {
    test.skip(testInfo.project.name === 'mobile', 'admin pages primarily desktop')
    await mockAdmin(page)
    await openEngineEditor(page)
    // Modal auto-focuses first input (B). Click modal title to defocus → A.
    await page.locator('.n-modal .n-card-header__main').click()
    await page.waitForTimeout(150)
    const stage = await page.locator('.si').first().getAttribute('data-stage')
    test.expect(stage).toBe('A')
    await page.screenshot({ path: shotPath('01-state-A-idle-shows-original', testInfo.project.name), fullPage: false })
  })

  test('02-state-B-focus-clears-shows-original-as-placeholder', async ({ page }, testInfo) => {
    test.skip(testInfo.project.name === 'mobile', 'admin pages primarily desktop')
    await mockAdmin(page)
    await openEngineEditor(page)
    const input = await nameInput(page)
    await input.focus()
    await page.waitForTimeout(150)
    const stage = await page.locator('.si').first().getAttribute('data-stage')
    test.expect(stage).toBe('B')
    // The input value should be empty in B; placeholder should be 'Google'
    const value = await input.inputValue()
    test.expect(value).toBe('')
    const placeholder = await input.getAttribute('placeholder')
    test.expect(placeholder).toBe('Google')
    await page.screenshot({ path: shotPath('02-state-B-focus-clears-shows-original-as-placeholder', testInfo.project.name), fullPage: false })
  })

  test('03-state-B-blur-without-input-restores', async ({ page }, testInfo) => {
    test.skip(testInfo.project.name === 'mobile', 'admin pages primarily desktop')
    await mockAdmin(page)
    await openEngineEditor(page)
    const input = await nameInput(page)
    await input.focus()
    await page.waitForTimeout(100)
    // Click on modal title to blur (still inside modal, doesn't close it)
    await page.locator('.n-modal .n-card-header__main').click()
    await page.waitForTimeout(150)
    const stage = await page.locator('.si').first().getAttribute('data-stage')
    test.expect(stage).toBe('A')
    const value = await input.inputValue()
    test.expect(value).toBe('Google')
    await page.screenshot({ path: shotPath('03-state-B-blur-without-input-restores', testInfo.project.name), fullPage: false })
  })

  test('04-state-D-typed-then-blur-shows-revert', async ({ page }, testInfo) => {
    test.skip(testInfo.project.name === 'mobile', 'admin pages primarily desktop')
    await mockAdmin(page)
    await openEngineEditor(page)
    const input = await nameInput(page)
    await input.focus()
    await input.fill('Bing')
    // Click outside input (modal title) to blur
    await page.locator('.n-modal .n-card-header__main').click()
    await page.waitForTimeout(150)
    const stage = await page.locator('.si').first().getAttribute('data-stage')
    test.expect(stage).toBe('D')
    const value = await input.inputValue()
    test.expect(value).toBe('Bing')
    // Revert button should be visible
    const revert = page.locator('.si').first().locator('.si__revert')
    await test.expect(revert).toBeVisible()
    await page.screenshot({ path: shotPath('04-state-D-typed-then-blur-shows-revert', testInfo.project.name), fullPage: false })
  })

  test('05-state-D-revert-click-restores-to-A', async ({ page }, testInfo) => {
    test.skip(testInfo.project.name === 'mobile', 'admin pages primarily desktop')
    await mockAdmin(page)
    await openEngineEditor(page)
    const input = await nameInput(page)
    await input.focus()
    await input.fill('Bing')
    await page.locator('.n-modal .n-card-header__main').click()
    await page.waitForTimeout(100)
    // Click revert button
    await page.locator('.si').first().locator('.si__revert').click()
    await page.waitForTimeout(150)
    const stage = await page.locator('.si').first().getAttribute('data-stage')
    test.expect(stage).toBe('A')
    const value = await input.inputValue()
    test.expect(value).toBe('Google')
    await page.screenshot({ path: shotPath('05-state-D-revert-click-restores-to-A', testInfo.project.name), fullPage: false })
  })

  test('06-state-D-empty-shows-highlighted-revert', async ({ page }, testInfo) => {
    test.skip(testInfo.project.name === 'mobile', 'admin pages primarily desktop')
    await mockAdmin(page)
    await openEngineEditor(page)
    const input = await nameInput(page)
    await input.focus()
    // Type something then clear it via select-all + delete
    await input.fill('xyz')
    await input.fill('')
    await page.locator('.n-modal .n-card-header__main').click()
    await page.waitForTimeout(150)
    const stage = await page.locator('.si').first().getAttribute('data-stage')
    test.expect(stage).toBe('D')
    // Highlighted revert (yellow pulse)
    const highlightCount = await page.locator('.si').first().locator('.si__revert--highlight').count()
    test.expect(highlightCount).toBe(1)
    await page.screenshot({ path: shotPath('06-state-D-empty-shows-highlighted-revert', testInfo.project.name), fullPage: false })
  })

  test('07-state-C-typing-active', async ({ page }, testInfo) => {
    test.skip(testInfo.project.name === 'mobile', 'admin pages primarily desktop')
    await mockAdmin(page)
    await openEngineEditor(page)
    const input = await nameInput(page)
    await input.focus()
    await input.type('B', { delay: 50 })
    // While focused with text typed → state C
    const stage = await page.locator('.si').first().getAttribute('data-stage')
    test.expect(stage).toBe('C')
    // Placeholder should be the original prop placeholder, not the original value
    const placeholder = await input.getAttribute('placeholder')
    test.expect(placeholder).toBe('例如：Google / 必应 / Searxng')
    await page.screenshot({ path: shotPath('07-state-C-typing-active', testInfo.project.name), fullPage: false })
  })

  test('08-empty-original-skips-B-goes-to-C', async ({ page }, testInfo) => {
    test.skip(testInfo.project.name === 'mobile', 'admin pages primarily desktop')
    await mockAdmin(page)
    // Open the CREATE editor (all originals empty)
    await page.goto('/admin/settings')
    await page.waitForSelector('.n-data-table-tr', { timeout: 10_000 })
    await page.locator('button', { hasText: '新建引擎' }).click()
    await page.waitForSelector('.n-modal')
    await page.waitForSelector('.si', { timeout: 5000 })
    const input = await nameInput(page)
    await input.focus()
    await page.waitForTimeout(150)
    const stage = await page.locator('.si').first().getAttribute('data-stage')
    // Empty originalValue → focus skips B, goes to C immediately
    test.expect(stage).toBe('C')
    await page.screenshot({ path: shotPath('08-empty-original-skips-B-goes-to-C', testInfo.project.name), fullPage: false })
  })
})
