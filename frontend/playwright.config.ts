import { defineConfig, devices } from '@playwright/test'

// E2E config for visual verification screenshots.
// All API calls are mocked via page.route() in spec files — no Go backend needed.
// `vite preview` serves the built dist; webServer below builds + starts it.
export default defineConfig({
  testDir: './tests/e2e',
  fullyParallel: false,
  forbidOnly: !!process.env.CI,
  retries: 0,
  workers: 1,
  reporter: 'list',

  use: {
    baseURL: 'http://localhost:4173',
    trace: 'off',
    video: 'off',
  },

  projects: [
    {
      name: 'desktop',
      use: {
        ...devices['Desktop Chrome'],
        viewport: { width: 1280, height: 800 },
      },
    },
    {
      name: 'mobile',
      // Use Chromium engine with iPhone viewport/UA — visual screenshots only.
      // Real WebKit isn't needed and would require an extra ~85MB browser install.
      use: {
        ...devices['iPhone 14 Pro Max'],
        browserName: 'chromium',
      },
    },
  ],

  webServer: {
    command: 'npm run build && npx vite preview --port 4173 --strictPort',
    url: 'http://localhost:4173',
    reuseExistingServer: !process.env.CI,
    timeout: 180_000,
  },
})
