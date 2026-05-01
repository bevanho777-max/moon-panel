# Moon Panel — frontend dev launcher (Windows / PowerShell)
#
# Usage from frontend/:
#   .\dev.ps1                 # vite dev server on :5173 (localhost only)
#   .\dev.ps1 -Lan             # bind 0.0.0.0 — phone/iPad can hit 192.168.x.x:5173
#                              # (avoid -Host name; collides with built-in $Host)
#
# Why this file:
# - One-line start matching backend/dev.ps1 for symmetric workflow
# - Auto-runs `npm install` if node_modules is missing (first-time setup)
# - The -Host flag is convenience for cross-device testing per docs/DEV.md §6.5

param(
  [switch]$Lan
)

$ErrorActionPreference = 'Stop'

if (-not (Test-Path './node_modules')) {
  Write-Host "node_modules missing — running 'npm install' (one-time)..." -ForegroundColor Yellow
  & npm install
  if ($LASTEXITCODE -ne 0) {
    Write-Host "npm install failed." -ForegroundColor Red
    exit $LASTEXITCODE
  }
}

Write-Host ""
Write-Host "Moon Panel dev frontend (Vite)" -ForegroundColor Cyan
Write-Host "  Port: 5173"
Write-Host "  Proxy: /api, /uploads, /assets/wallpapers -> http://localhost:3001"
if ($Lan) { Write-Host "  Network: 0.0.0.0 (LAN-accessible)" }
Write-Host ""

if ($Lan) {
  & npm run dev -- --host
} else {
  & npm run dev
}
