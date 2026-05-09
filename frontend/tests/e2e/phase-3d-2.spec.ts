import { test, type Page } from '@playwright/test'
import path from 'node:path'

// Phase 3d-2: audit log + login lockout + session expiry UX.
// Independent spec — covers the new admin AuditLog page, the login screen's
// 记住我 checkbox, the ?reason=expired banner, and the 429 lockout error.

const SHOTS_BASE = process.env.MOON_SHOTS_DIR ?? 'P:/moon-panel/screenshots/phase-3d-2'

const NOW = '2026-04-30T12:00:00+08:00'

function ts(offsetSec: number): string {
  return new Date(Date.parse(NOW) + offsetSec * 1000).toISOString()
}

const SAMPLE_AUDIT_ITEMS = [
  {
    id: 1042,
    timestamp: ts(0),
    actor: 'admin',
    action: 'login_success',
    target_type: '',
    target_id: '',
    ip: '203.0.113.45',
    user_agent: 'Mozilla/5.0 (Macintosh; Intel Mac OS X 14_2) AppleWebKit/537.36',
    status: 200,
    details: JSON.stringify({ remember_me: true, ttl_seconds: 2592000 }),
    created_at: ts(0),
  },
  {
    id: 1041,
    timestamp: ts(-30),
    actor: 'admin',
    action: 'PUT /api/admin/cards/3',
    target_type: '',
    target_id: '',
    ip: '203.0.113.45',
    user_agent: 'Mozilla/5.0 (Macintosh; Intel Mac OS X 14_2) AppleWebKit/537.36',
    status: 200,
    details: JSON.stringify({
      method: 'PUT',
      path: '/api/admin/cards/3',
      body: { title: 'Plex (renamed)', description: '本地影音' },
    }),
    created_at: ts(-30),
  },
  {
    id: 1040,
    timestamp: ts(-95),
    actor: 'admin',
    action: 'password_change',
    target_type: '',
    target_id: '',
    ip: '203.0.113.45',
    user_agent: 'Mozilla/5.0 (Macintosh; Intel Mac OS X 14_2) AppleWebKit/537.36',
    status: 200,
    details: '{}',
    created_at: ts(-95),
  },
  {
    id: 1039,
    timestamp: ts(-160),
    actor: 'admin',
    action: 'POST /api/admin/cards',
    target_type: '',
    target_id: '',
    ip: '203.0.113.45',
    user_agent: 'Mozilla/5.0 (Macintosh; Intel Mac OS X 14_2) AppleWebKit/537.36',
    status: 200,
    details: JSON.stringify({
      method: 'POST',
      path: '/api/admin/cards',
      body: { title: 'Vaultwarden', icon: 'lucide:shield', url_external: 'https://vault.example.com' },
    }),
    created_at: ts(-160),
  },
  {
    id: 1038,
    timestamp: ts(-340),
    actor: 'admin',
    action: 'DELETE /api/admin/cards/7',
    target_type: '',
    target_id: '',
    ip: '203.0.113.45',
    user_agent: 'Mozilla/5.0 (Macintosh; Intel Mac OS X 14_2) AppleWebKit/537.36',
    status: 200,
    details: JSON.stringify({ method: 'DELETE', path: '/api/admin/cards/7' }),
    created_at: ts(-340),
  },
  {
    id: 1037,
    timestamp: ts(-7300),
    actor: 'unknown',
    action: 'login_failure',
    target_type: '',
    target_id: '',
    ip: '198.51.100.22',
    user_agent: 'curl/7.81.0',
    status: 401,
    details: JSON.stringify({ username_tried: 'root', reason: 'user_not_found' }),
    created_at: ts(-7300),
  },
  {
    id: 1036,
    timestamp: ts(-7301),
    actor: 'admin',
    action: 'login_failure',
    target_type: '',
    target_id: '',
    ip: '198.51.100.22',
    user_agent: 'curl/7.81.0',
    status: 401,
    details: JSON.stringify({ username_tried: 'admin', reason: 'password_mismatch' }),
    created_at: ts(-7301),
  },
]

async function mockAdmin(page: Page) {
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
}

async function mockAuditList(page: Page, items = SAMPLE_AUDIT_ITEMS, total = items.length) {
  await page.route('**/api/admin/audit-logs**', (route) =>
    route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({
        code: 0, msg: 'ok',
        data: { items, total, page: 1, size: 20 },
      }),
    }),
  )
}

function shotPath(name: string, project: string): string {
  return path.join(SHOTS_BASE, project, `${name}.png`)
}

test.describe('Phase 3d-2 audit log + lockout + session UX', () => {
  test('01-audit-log-page-loaded', async ({ page }, testInfo) => {
    test.skip(testInfo.project.name === 'mobile', 'admin pages primarily desktop')
    await mockAdmin(page)
    await mockAuditList(page)
    await page.goto('/admin/audit-logs')
    // v0.2.17: NaiveUI internal .n-data-table-tr -> own BEM .al__table--desktop tbody tr
    // (V2 lesson application, prevents NaiveUI upgrade staleness)
    await page.waitForSelector('.al__table--desktop tbody tr', { timeout: 10_000 })
    await page.waitForTimeout(200)
    await page.screenshot({ path: shotPath('01-audit-log-page-loaded', testInfo.project.name), fullPage: true })
  })

  test('02-audit-log-detail-modal', async ({ page }, testInfo) => {
    test.skip(testInfo.project.name === 'mobile', 'admin pages primarily desktop')
    await mockAdmin(page)
    await mockAuditList(page)
    await page.goto('/admin/audit-logs')
    // v0.2.17: NaiveUI internal -> own BEM (V2 lesson)
    await page.waitForSelector('.al__table--desktop tbody tr', { timeout: 10_000 })
    // Click "查看" on the second row (the PUT card update — has a body to display)
    await page.locator('.al__table--desktop tbody tr').nth(1).locator('button:has-text("查看")').click()
    await page.waitForSelector('.n-modal')
    await page.waitForTimeout(200)
    await page.screenshot({ path: shotPath('02-audit-log-detail-modal', testInfo.project.name), fullPage: false })
  })

  test('03-audit-log-action-filter-open', async ({ page }, testInfo) => {
    test.skip(testInfo.project.name === 'mobile', 'admin pages primarily desktop')
    await mockAdmin(page)
    await mockAuditList(page)
    await page.goto('/admin/audit-logs')
    // v0.2.17: NaiveUI internal -> own BEM (V2 lesson)
    await page.waitForSelector('.al__table--desktop tbody tr', { timeout: 10_000 })
    // Open the action filter dropdown
    await page.locator('.n-base-selection').first().click()
    await page.waitForSelector('.n-base-select-option', { timeout: 5000 })
    await page.waitForTimeout(200)
    await page.screenshot({ path: shotPath('03-audit-log-action-filter-open', testInfo.project.name), fullPage: false })
  })

  test('04-login-page-with-expired-banner', async ({ page }, testInfo) => {
    await page.route('**/api/auth/me', (route) =>
      route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          code: 0, msg: 'ok',
          data: { initialized: true, authenticated: false },
        }),
      }),
    )
    await page.goto('/login?reason=expired&redirect=%2Fadmin%2Fcards')
    await page.waitForSelector('.login__card', { timeout: 5000 })
    await page.waitForTimeout(200)
    await page.screenshot({ path: shotPath('04-login-page-with-expired-banner', testInfo.project.name), fullPage: false })
  })

  test('05-login-page-with-remember-me', async ({ page }, testInfo) => {
    await page.route('**/api/auth/me', (route) =>
      route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          code: 0, msg: 'ok',
          data: { initialized: true, authenticated: false },
        }),
      }),
    )
    await page.goto('/login')
    await page.waitForSelector('.login__card', { timeout: 5000 })
    // Check the 记住我 box for visual confirmation
    await page.locator('.n-checkbox').click()
    await page.waitForTimeout(200)
    await page.screenshot({ path: shotPath('05-login-page-with-remember-me', testInfo.project.name), fullPage: false })
  })

  test('06-login-lockout-429', async ({ page }, testInfo) => {
    test.skip(testInfo.project.name === 'mobile', 'visually identical')
    await page.route('**/api/auth/me', (route) =>
      route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          code: 0, msg: 'ok',
          data: { initialized: true, authenticated: false },
        }),
      }),
    )
    // Mock the lockout 429 response — what user sees after 5 failures.
    await page.route('**/api/auth/login', (route) =>
      route.fulfill({
        status: 429,
        contentType: 'application/json',
        headers: { 'Retry-After': '30m0s' },
        body: JSON.stringify({ code: 429, msg: 'too many failed login attempts; locked for 30m0s' }),
      }),
    )
    await page.goto('/login')
    await page.waitForSelector('.login__card', { timeout: 5000 })
    await page.locator('input[type="password"]').first().fill('wrong-password')
    await page.locator('button:has-text("登录")').click()
    // Wait for the message toast to appear
    await page.waitForSelector('.n-message', { timeout: 3000 })
    await page.waitForTimeout(200)
    await page.screenshot({ path: shotPath('06-login-lockout-429', testInfo.project.name), fullPage: false })
  })

  test('07-admin-menu-includes-audit-log', async ({ page }, testInfo) => {
    test.skip(testInfo.project.name === 'mobile', 'admin pages primarily desktop')
    await mockAdmin(page)
    await mockAuditList(page, [], 0)
    await page.goto('/admin')
    await page.waitForSelector('.admin-header__menu', { timeout: 10_000 })
    await page.waitForTimeout(200)
    await page.screenshot({ path: shotPath('07-admin-menu-includes-audit-log', testInfo.project.name), fullPage: false })
  })

  test('08-audit-log-empty-state', async ({ page }, testInfo) => {
    test.skip(testInfo.project.name === 'mobile', 'admin pages primarily desktop')
    await mockAdmin(page)
    await mockAuditList(page, [], 0)
    await page.goto('/admin/audit-logs')
    // v0.2.17: NaiveUI .n-data-table -> own BEM .al__table--desktop wrapper
    // (empty state, tbody has no tr — wait for table container only)
    await page.waitForSelector('.al__table--desktop', { timeout: 10_000 })
    await page.waitForTimeout(300)
    await page.screenshot({ path: shotPath('08-audit-log-empty-state', testInfo.project.name), fullPage: false })
  })
})
