import { test, type Page } from '@playwright/test'
import path from 'node:path'

// Phase 3d-1 hotfix: change-password UI
// Scope: dropdown user menu in admin header + ChangePasswordModal flow.
// Independent spec so 3c-2 baseline (108 cities, 19 desktop shots) stays clean.

const SHOTS_BASE = process.env.MOON_SHOTS_DIR ?? 'P:/moon-panel/screenshots/phase-3d-1'

const NOW = '2026-04-30T12:00:00+08:00'

async function mockAuthedAdmin(page: Page) {
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
  // Admin layout pulls these too — stub harmlessly so the page renders.
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
}

function shotPath(name: string, project: string): string {
  return path.join(SHOTS_BASE, project, `${name}.png`)
}

test.describe('Phase 3d-1 change-password UI', () => {
  test('01-admin-user-dropdown-open', async ({ page }, testInfo) => {
    test.skip(testInfo.project.name === 'mobile', 'admin pages primarily desktop')
    await mockAuthedAdmin(page)
    await page.goto('/admin')
    await page.waitForSelector('[data-testid="admin-user-menu"]', { timeout: 10_000 })
    await page.locator('[data-testid="admin-user-menu"]').click()
    await page.waitForSelector('.n-dropdown-option')
    await page.waitForTimeout(150)
    await page.screenshot({ path: shotPath('01-admin-user-dropdown-open', testInfo.project.name), fullPage: false })
  })

  test('02-change-password-modal-empty', async ({ page }, testInfo) => {
    test.skip(testInfo.project.name === 'mobile', 'admin pages primarily desktop')
    await mockAuthedAdmin(page)
    await page.goto('/admin')
    await page.waitForSelector('[data-testid="admin-user-menu"]', { timeout: 10_000 })
    await page.locator('[data-testid="admin-user-menu"]').click()
    await page.locator('.n-dropdown-option').filter({ hasText: '修改密码' }).click()
    await page.waitForSelector('.n-modal')
    await page.waitForTimeout(150)
    await page.screenshot({ path: shotPath('02-change-password-modal-empty', testInfo.project.name), fullPage: false })
  })

  test('03-change-password-strength-medium', async ({ page }, testInfo) => {
    test.skip(testInfo.project.name === 'mobile', 'admin pages primarily desktop')
    await mockAuthedAdmin(page)
    await page.goto('/admin')
    await page.locator('[data-testid="admin-user-menu"]').click()
    await page.locator('.n-dropdown-option').filter({ hasText: '修改密码' }).click()
    await page.waitForSelector('.n-modal')
    // Fill old + a 10-char letters+digits new password → "中" (medium)
    const inputs = page.locator('.n-modal input[type="password"]')
    await inputs.nth(0).fill('oldpass123')
    await inputs.nth(1).fill('mediumpw12') // 10 chars, letter+digit → medium
    await inputs.nth(2).fill('mediumpw12')
    await page.waitForTimeout(200)
    await page.screenshot({ path: shotPath('03-change-password-strength-medium', testInfo.project.name), fullPage: false })
  })

  test('04-change-password-strength-strong', async ({ page }, testInfo) => {
    test.skip(testInfo.project.name === 'mobile', 'admin pages primarily desktop')
    await mockAuthedAdmin(page)
    await page.goto('/admin')
    await page.locator('[data-testid="admin-user-menu"]').click()
    await page.locator('.n-dropdown-option').filter({ hasText: '修改密码' }).click()
    await page.waitForSelector('.n-modal')
    const inputs = page.locator('.n-modal input[type="password"]')
    await inputs.nth(0).fill('oldpass123')
    await inputs.nth(1).fill('Str0ng!Passw0rd2026') // 19 chars + 3 classes → strong
    await inputs.nth(2).fill('Str0ng!Passw0rd2026')
    await page.waitForTimeout(200)
    await page.screenshot({ path: shotPath('04-change-password-strength-strong', testInfo.project.name), fullPage: false })
  })

  test('05-change-password-mismatch-hint', async ({ page }, testInfo) => {
    test.skip(testInfo.project.name === 'mobile', 'admin pages primarily desktop')
    await mockAuthedAdmin(page)
    await page.goto('/admin')
    await page.locator('[data-testid="admin-user-menu"]').click()
    await page.locator('.n-dropdown-option').filter({ hasText: '修改密码' }).click()
    await page.waitForSelector('.n-modal')
    const inputs = page.locator('.n-modal input[type="password"]')
    await inputs.nth(0).fill('oldpass123')
    await inputs.nth(1).fill('Str0ng!Passw0rd2026')
    await inputs.nth(2).fill('different-confirm')
    await page.waitForTimeout(200)
    await page.screenshot({ path: shotPath('05-change-password-mismatch-hint', testInfo.project.name), fullPage: false })
  })

  test('06-change-password-old-incorrect', async ({ page }, testInfo) => {
    test.skip(testInfo.project.name === 'mobile', 'admin pages primarily desktop')
    await mockAuthedAdmin(page)
    // Stub the PUT to return 401 — simulates user typing wrong old password.
    await page.route('**/api/auth/password', (route) =>
      route.fulfill({
        status: 401,
        contentType: 'application/json',
        body: JSON.stringify({ code: 401, msg: 'old password incorrect' }),
      }),
    )
    await page.goto('/admin')
    await page.locator('[data-testid="admin-user-menu"]').click()
    await page.locator('.n-dropdown-option').filter({ hasText: '修改密码' }).click()
    await page.waitForSelector('.n-modal')
    const inputs = page.locator('.n-modal input[type="password"]')
    await inputs.nth(0).fill('wrong-old-password')
    await inputs.nth(1).fill('Str0ng!Passw0rd2026')
    await inputs.nth(2).fill('Str0ng!Passw0rd2026')
    await page.locator('.n-modal button').filter({ hasText: '确认修改' }).click()
    // Wait for inline error alert to appear
    await page.waitForSelector('.n-alert', { timeout: 3000 })
    await page.waitForTimeout(200)
    await page.screenshot({ path: shotPath('06-change-password-old-incorrect', testInfo.project.name), fullPage: false })
  })
})
