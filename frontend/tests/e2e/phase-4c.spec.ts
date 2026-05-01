import { test, type Page } from '@playwright/test'
import path from 'node:path'

// Phase 4c: backup/restore UI + StatefulAutoComplete + IconAutoComplete
// reuse to Groups + first-visit toast hint.

const SHOTS_BASE = process.env.MOON_SHOTS_DIR ?? 'P:/moon-panel/screenshots/phase-4c'
const NOW = '2026-05-01T12:00:00+08:00'

const SAMPLE_GROUPS = [
  { id: 1, name: 'Media', icon: '', sort: 10, created_at: NOW, updated_at: NOW },
]
const SAMPLE_CARD = {
  id: 1,
  group_id: 1,
  title: 'Jellyfin',
  description: 'Local media',
  icon: 'lucide:film',
  icon_type: 'url',
  url_internal: 'http://j.lan',
  url_external: '',
  url_default: 'internal',
  open_in_new_tab: true,
  sort: 10,
  created_at: NOW,
  updated_at: NOW,
}

async function mockAdmin(page: Page) {
  await page.route('**/api/auth/me', (route) =>
    route.fulfill({
      status: 200, contentType: 'application/json',
      body: JSON.stringify({
        code: 0, msg: 'ok',
        data: { initialized: true, authenticated: true, username: 'admin', totp_enabled: false },
      }),
    }),
  )
  await page.route('**/api/admin/search-engines', (route) =>
    route.fulfill({
      status: 200, contentType: 'application/json',
      body: JSON.stringify({ code: 0, msg: 'ok', data: [] }),
    }),
  )
  await page.route('**/api/admin/settings', (route) =>
    route.fulfill({
      status: 200, contentType: 'application/json',
      body: JSON.stringify({ code: 0, msg: 'ok', data: { 'widget.cities': '[]', 'widget.temp_unit': 'C' } }),
    }),
  )
  await page.route('**/api/admin/security/trusted-ips', (route) =>
    route.fulfill({
      status: 200, contentType: 'application/json',
      body: JSON.stringify({ code: 0, msg: 'ok', data: { items: [] } }),
    }),
  )
  await page.route('**/api/admin/groups', (route) =>
    route.fulfill({
      status: 200, contentType: 'application/json',
      body: JSON.stringify({ code: 0, msg: 'ok', data: SAMPLE_GROUPS }),
    }),
  )
  await page.route('**/api/admin/cards', (route) =>
    route.fulfill({
      status: 200, contentType: 'application/json',
      body: JSON.stringify({ code: 0, msg: 'ok', data: [SAMPLE_CARD] }),
    }),
  )
  // Cards.vue openEdit calls GET /admin/cards/:id for fresh data
  await page.route('**/api/admin/cards/1', (route) =>
    route.fulfill({
      status: 200, contentType: 'application/json',
      body: JSON.stringify({ code: 0, msg: 'ok', data: SAMPLE_CARD }),
    }),
  )
}

function shotPath(name: string, project: string): string {
  return path.join(SHOTS_BASE, project, `${name}.png`)
}

test.describe('Phase 4c backup + StatefulAutoComplete + IconAutoComplete reuse', () => {
  test('01-backup-section-in-settings', async ({ page }, testInfo) => {
    test.skip(testInfo.project.name === 'mobile', 'admin pages primarily desktop')
    await mockAdmin(page)
    await page.goto('/admin/settings')
    await page.locator('text=备份与恢复').waitFor({ timeout: 10_000 })
    // Scroll to bottom to make backup section visible
    await page.evaluate(() => {
      const card = Array.from(document.querySelectorAll('.n-card')).find((c) =>
        c.textContent?.includes('备份与恢复'),
      )
      card?.scrollIntoView({ block: 'center' })
    })
    await page.waitForTimeout(200)
    await page.screenshot({ path: shotPath('01-backup-section-in-settings', testInfo.project.name), fullPage: false })
  })

  test('02-restore-modal-stage-select', async ({ page }, testInfo) => {
    test.skip(testInfo.project.name === 'mobile', 'admin pages primarily desktop')
    await mockAdmin(page)
    await page.goto('/admin/settings')
    await page.locator('button', { hasText: '从备份恢复' }).waitFor({ timeout: 10_000 })
    await page.locator('button', { hasText: '从备份恢复' }).click()
    await page.waitForSelector('.n-modal')
    await page.waitForTimeout(200)
    await page.screenshot({ path: shotPath('02-restore-modal-stage-select', testInfo.project.name), fullPage: false })
  })

  test('03-icon-autocomplete-on-groups', async ({ page }, testInfo) => {
    test.skip(testInfo.project.name === 'mobile', 'admin pages primarily desktop')
    await mockAdmin(page)
    await page.goto('/admin/groups')
    await page.locator('button', { hasText: '编辑' }).first().waitFor({ timeout: 10_000 })
    // Pre-set the localStorage flag so the toast doesn't pop up and obscure the modal.
    await page.evaluate(() => localStorage.setItem('moon.statefulInputHintShown', 'true'))
    await page.locator('button', { hasText: '编辑' }).first().click()
    await page.waitForSelector('.n-modal')
    // Focus the icon field — should trigger catalog load (lazy chunk)
    await page.locator('.n-modal input').nth(1).focus()
    await page.waitForTimeout(800) // give async catalog chunks time to load
    await page.locator('.n-modal input').nth(1).fill('jelly')
    await page.waitForTimeout(500)
    await page.screenshot({ path: shotPath('03-icon-autocomplete-on-groups', testInfo.project.name), fullPage: false })
  })

  test('04-icon-autocomplete-on-cards-with-suggestions', async ({ page }, testInfo) => {
    test.skip(testInfo.project.name === 'mobile', 'admin pages primarily desktop')
    await mockAdmin(page)
    await page.goto('/admin/cards')
    await page.locator('button', { hasText: '编辑' }).first().waitFor({ timeout: 10_000 })
    await page.evaluate(() => localStorage.setItem('moon.statefulInputHintShown', 'true'))
    await page.locator('button', { hasText: '编辑' }).first().click()
    await page.waitForSelector('.n-modal')
    // Find the IconAutoComplete by its sac wrapper (StatefulAutoComplete root)
    const iconInput = page.locator('.sac input').first()
    await iconInput.scrollIntoViewIfNeeded()
    await iconInput.focus()
    await page.waitForTimeout(800)
    await iconInput.fill('plex')
    await page.waitForTimeout(500)
    await page.screenshot({ path: shotPath('04-icon-autocomplete-on-cards-with-suggestions', testInfo.project.name), fullPage: false })
  })

  test('05-first-visit-hint-toast', async ({ page }, testInfo) => {
    test.skip(testInfo.project.name === 'mobile', 'admin pages primarily desktop')
    await mockAdmin(page)
    await page.goto('/admin/groups')
    await page.locator('button', { hasText: '编辑' }).first().waitFor({ timeout: 10_000 })
    // Ensure flag is NOT set (fresh user)
    await page.evaluate(() => localStorage.removeItem('moon.statefulInputHintShown'))
    await page.locator('button', { hasText: '编辑' }).first().click()
    // Wait for the toast to appear (NaiveUI .n-message)
    await page.waitForSelector('.n-message', { timeout: 3000 })
    await page.waitForTimeout(200)
    await page.screenshot({ path: shotPath('05-first-visit-hint-toast', testInfo.project.name), fullPage: false })
  })

  test('06-stateful-autocomplete-state-D-revert', async ({ page }, testInfo) => {
    test.skip(testInfo.project.name === 'mobile', 'admin pages primarily desktop')
    await mockAdmin(page)
    await page.goto('/admin/cards')
    await page.locator('button', { hasText: '编辑' }).first().waitFor({ timeout: 10_000 })
    await page.evaluate(() => localStorage.setItem('moon.statefulInputHintShown', 'true'))
    await page.locator('button', { hasText: '编辑' }).first().click()
    await page.waitForSelector('.n-modal')
    const iconInput = page.locator('.n-form-item:has-text("图标") input').first()
    await iconInput.focus()
    await page.waitForTimeout(300)
    await iconInput.fill('changed-icon-name')
    // Click outside to blur — go to title which is a different field
    await page.locator('.n-modal').locator('text=标题').first().click()
    await page.waitForTimeout(300)
    // Should now see ↺ revert button on the icon field
    await page.screenshot({ path: shotPath('06-stateful-autocomplete-state-D-revert', testInfo.project.name), fullPage: false })
  })
})
