import { test, type Page } from '@playwright/test'
import path from 'node:path'

// Phase 3d-3: TOTP / 2FA enrollment + login two-step.
// Independent spec — exercises the new admin enroll flow, the disable modal,
// the login screen's TOTP step, and the audit page's new friendly labels.

const SHOTS_BASE = process.env.MOON_SHOTS_DIR ?? 'P:/moon-panel/screenshots/phase-3d-3'

const NOW = '2026-05-01T12:00:00+08:00'

const SAMPLE_ENROLL = {
  secret: 'JBSWY3DPEHPK3PXPK5VBVKQQVUSIYXAB',
  otpauth_url:
    'otpauth://totp/Moon%20Panel:admin?secret=JBSWY3DPEHPK3PXPK5VBVKQQVUSIYXAB&issuer=Moon%20Panel&algorithm=SHA1&digits=6&period=30',
  backup_codes: [
    'A3BC-8X9F', 'KP2M-QH7N', 'WZ4D-LR2T', 'YH9V-NX8P',
    '5JBS-WY3D', 'PEHP-K3PX', '6Q8M-RTVB', 'F1JK-LMNZ',
  ],
}

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
      body: JSON.stringify({
        code: 0, msg: 'ok',
        data: { 'widget.cities': '[]', 'widget.temp_unit': 'C' },
      }),
    }),
  )
}

function shotPath(name: string, project: string): string {
  return path.join(SHOTS_BASE, project, `${name}.png`)
}

test.describe('Phase 3d-3 2FA / TOTP', () => {
  test('01-settings-section-disabled', async ({ page }, testInfo) => {
    test.skip(testInfo.project.name === 'mobile', 'admin pages primarily desktop')
    await mockAdmin(page, false)
    await page.goto('/admin/settings')
    await page.waitForSelector('.n-card', { timeout: 10_000 })
    // Wait for the 2FA section to render.
    await page.locator('button', { hasText: '启用两步验证' }).waitFor({ timeout: 5000 })
    await page.evaluate(() => {
      const card = Array.from(document.querySelectorAll('.n-card')).find((c) =>
        c.textContent?.includes('两步验证'),
      )
      card?.scrollIntoView({ block: 'center' })
    })
    await page.waitForTimeout(200)
    await page.screenshot({ path: shotPath('01-settings-section-disabled', testInfo.project.name), fullPage: false })
  })

  test('02-enroll-modal-scan-stage', async ({ page }, testInfo) => {
    test.skip(testInfo.project.name === 'mobile', 'admin pages primarily desktop')
    await mockAdmin(page, false)
    await page.route('**/api/auth/2fa/enroll', (route) =>
      route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({ code: 0, msg: 'ok', data: SAMPLE_ENROLL }),
      }),
    )
    await page.goto('/admin/settings')
    await page.locator('button', { hasText: '启用两步验证' }).waitFor({ timeout: 10_000 })
    await page.locator('button', { hasText: '启用两步验证' }).click()
    // Wait for QR image to appear (qrcode lib renders to data URL)
    await page.waitForSelector('.t2f__qr', { timeout: 5000 })
    await page.waitForTimeout(300)
    await page.screenshot({ path: shotPath('02-enroll-modal-scan-stage', testInfo.project.name), fullPage: false })
  })

  test('03-enroll-modal-backup-codes-stage', async ({ page }, testInfo) => {
    test.skip(testInfo.project.name === 'mobile', 'admin pages primarily desktop')
    await mockAdmin(page, false)
    await page.route('**/api/auth/2fa/enroll', (route) =>
      route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({ code: 0, msg: 'ok', data: SAMPLE_ENROLL }),
      }),
    )
    await page.route('**/api/auth/2fa/confirm', (route) =>
      route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({ code: 0, msg: 'ok', data: { ok: true } }),
      }),
    )
    await page.goto('/admin/settings')
    await page.locator('button', { hasText: '启用两步验证' }).waitFor({ timeout: 10_000 })
    await page.locator('button', { hasText: '启用两步验证' }).click()
    await page.waitForSelector('.t2f__qr', { timeout: 5000 })
    // Type confirm code to advance to backup-codes stage
    await page.locator('.n-modal input').first().fill('123456')
    await page.locator('.n-modal button', { hasText: '确认启用' }).click()
    await page.waitForSelector('.t2f__codes', { timeout: 5000 })
    await page.waitForTimeout(200)
    await page.screenshot({ path: shotPath('03-enroll-modal-backup-codes-stage', testInfo.project.name), fullPage: false })
  })

  test('04-settings-section-enabled', async ({ page }, testInfo) => {
    test.skip(testInfo.project.name === 'mobile', 'admin pages primarily desktop')
    await mockAdmin(page, true)
    await page.goto('/admin/settings')
    await page.locator('button', { hasText: '禁用两步验证' }).waitFor({ timeout: 10_000 })
    await page.evaluate(() => {
      const card = Array.from(document.querySelectorAll('.n-card')).find((c) =>
        c.textContent?.includes('两步验证'),
      )
      card?.scrollIntoView({ block: 'center' })
    })
    await page.waitForTimeout(200)
    await page.screenshot({ path: shotPath('04-settings-section-enabled', testInfo.project.name), fullPage: false })
  })

  test('05-disable-modal', async ({ page }, testInfo) => {
    test.skip(testInfo.project.name === 'mobile', 'admin pages primarily desktop')
    await mockAdmin(page, true)
    await page.goto('/admin/settings')
    await page.locator('button', { hasText: '禁用两步验证' }).waitFor({ timeout: 10_000 })
    await page.locator('button', { hasText: '禁用两步验证' }).click()
    await page.waitForSelector('.n-modal')
    await page.waitForTimeout(200)
    await page.screenshot({ path: shotPath('05-disable-modal', testInfo.project.name), fullPage: false })
  })

  test('06-login-totp-step', async ({ page }, testInfo) => {
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
    await page.route('**/api/auth/login', (route) =>
      route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({ code: 0, msg: 'ok', data: { needs_2fa: true } }),
      }),
    )
    await page.goto('/login')
    await page.waitForSelector('.login__card')
    await page.locator('input[type="password"]').first().fill('correct-password')
    await page.locator('button', { hasText: '登录' }).click()
    // Should now show TOTP step
    await page.waitForSelector('[data-testid="totp-input"]', { timeout: 5000 })
    await page.waitForTimeout(200)
    await page.screenshot({ path: shotPath('06-login-totp-step', testInfo.project.name), fullPage: false })
  })

  test('07-login-totp-failed', async ({ page }, testInfo) => {
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
    await page.route('**/api/auth/login', (route) =>
      route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({ code: 0, msg: 'ok', data: { needs_2fa: true } }),
      }),
    )
    await page.route('**/api/auth/2fa/verify', (route) =>
      route.fulfill({
        status: 401,
        contentType: 'application/json',
        body: JSON.stringify({ code: 401, msg: 'code does not match' }),
      }),
    )
    await page.goto('/login')
    await page.locator('input[type="password"]').first().fill('correct-password')
    await page.locator('button', { hasText: '登录' }).click()
    await page.waitForSelector('[data-testid="totp-input"]', { timeout: 5000 })
    await page.locator('[data-testid="totp-input"] input').fill('000000')
    await page.locator('button', { hasText: '确认' }).click()
    await page.waitForSelector('.n-alert', { timeout: 3000 })
    await page.waitForTimeout(200)
    await page.screenshot({ path: shotPath('07-login-totp-failed', testInfo.project.name), fullPage: false })
  })

  test('08-login-backup-code-mode', async ({ page }, testInfo) => {
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
    await page.route('**/api/auth/login', (route) =>
      route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({ code: 0, msg: 'ok', data: { needs_2fa: true } }),
      }),
    )
    await page.goto('/login')
    await page.locator('input[type="password"]').first().fill('correct-password')
    await page.locator('button', { hasText: '登录' }).click()
    await page.waitForSelector('[data-testid="totp-input"]', { timeout: 5000 })
    // Switch to backup code mode
    await page.locator('button', { hasText: '改用备份码' }).click()
    await page.waitForTimeout(200)
    await page.screenshot({ path: shotPath('08-login-backup-code-mode', testInfo.project.name), fullPage: false })
  })

  test('09-audit-log-with-totp-actions', async ({ page }, testInfo) => {
    test.skip(testInfo.project.name === 'mobile', 'admin pages primarily desktop')
    const ts = (offset: number) => new Date(Date.parse(NOW) + offset * 1000).toISOString()
    const items = [
      {
        id: 2001, timestamp: ts(0), actor: 'admin', action: 'totp_verify_success',
        target_type: '', target_id: '', ip: '203.0.113.45', user_agent: 'Chrome',
        status: 200, details: JSON.stringify({ via_backup: false, ttl_seconds: 604800 }),
        created_at: ts(0),
      },
      {
        id: 2000, timestamp: ts(-30), actor: 'admin', action: 'login_password_ok_awaiting_2fa',
        target_type: '', target_id: '', ip: '203.0.113.45', user_agent: 'Chrome',
        status: 200, details: '{}',
        created_at: ts(-30),
      },
      {
        id: 1999, timestamp: ts(-120), actor: 'admin', action: 'totp_verify_failure',
        target_type: '', target_id: '', ip: '198.51.100.22', user_agent: 'curl/7.81',
        status: 401, details: JSON.stringify({ reason: 'code_mismatch', is_backup: false }),
        created_at: ts(-120),
      },
      {
        id: 1998, timestamp: ts(-360), actor: 'admin', action: 'backup_code_used',
        target_type: '', target_id: '', ip: '203.0.113.45', user_agent: 'iPhone',
        status: 200, details: JSON.stringify({ remaining: 7 }),
        created_at: ts(-360),
      },
      {
        id: 1997, timestamp: ts(-3600), actor: 'admin', action: 'totp_enable',
        target_type: '', target_id: '', ip: '192.168.1.10', user_agent: 'Chrome',
        status: 200, details: '{}',
        created_at: ts(-3600),
      },
    ]
    await page.route('**/api/auth/me', (route) =>
      route.fulfill({
        status: 200, contentType: 'application/json',
        body: JSON.stringify({
          code: 0, msg: 'ok',
          data: { initialized: true, authenticated: true, username: 'admin', totp_enabled: true },
        }),
      }),
    )
    await page.route('**/api/admin/audit-logs**', (route) =>
      route.fulfill({
        status: 200, contentType: 'application/json',
        body: JSON.stringify({ code: 0, msg: 'ok', data: { items, total: items.length, page: 1, size: 20 } }),
      }),
    )
    await page.goto('/admin/audit-logs')
    await page.waitForSelector('.n-data-table-tr', { timeout: 10_000 })
    await page.waitForTimeout(200)
    await page.screenshot({ path: shotPath('09-audit-log-with-totp-actions', testInfo.project.name), fullPage: true })
  })
})
