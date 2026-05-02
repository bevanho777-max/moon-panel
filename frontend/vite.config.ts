/// <reference types="vitest" />
import { defineConfig } from 'vite'
import vue from '@vitejs/plugin-vue'
import { fileURLToPath, URL } from 'node:url'

export default defineConfig({
  plugins: [vue()],
  // Strip console.* and `debugger` from the production bundle. Vite's
  // default esbuild minify keeps console calls; for a public-facing panel
  // we don't want diagnostic spam in users' browser consoles. Marginal
  // bundle-size + INP win (~5ms tops); the main motivation is hygiene.
  esbuild: {
    drop: ['console', 'debugger'],
  },
  resolve: {
    alias: {
      '@': fileURLToPath(new URL('./src', import.meta.url)),
    },
  },
  server: {
    port: 5173,
    // Local dev: Vite serves frontend at :5173 and proxies API + static
    // backend resources to a Go server at :3001 (one port above the
    // production :3000 so a NAS-deployed prod container can keep running
    // alongside dev). Three proxy roots:
    //   /api/*               — REST endpoints (auth, panel, settings, ...)
    //   /uploads/*           — user-uploaded icons + wallpapers
    //   /assets/wallpapers/* — go:embed builtin wallpaper SVGs
    // Note we DO NOT proxy /assets/* broadly — Vite uses /assets/ for its
    // own dev-time module URLs, which would conflict.
    proxy: {
      '/api': { target: 'http://localhost:3001', changeOrigin: false },
      '/uploads': { target: 'http://localhost:3001', changeOrigin: false },
      '/assets/wallpapers': { target: 'http://localhost:3001', changeOrigin: false },
    },
  },
  build: {
    outDir: '../backend/web/dist',
    emptyOutDir: true,
  },
  test: {
    environment: 'node',
    include: ['src/**/*.{spec,test}.ts'],
    // Keep Playwright e2e specs out of vitest's scope.
    exclude: ['node_modules/**', 'tests/e2e/**'],
  },
})
