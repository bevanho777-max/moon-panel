# Moon Panel — backend dev launcher (Windows / PowerShell)
#
# Usage from backend/:
#   .\dev.ps1                 # default: air hot-reload
#   .\dev.ps1 -NoAir          # plain `go run` (no hot reload)
#   .\dev.ps1 -Port 3002      # override port
#   .\dev.ps1 -Pw mypw1234    # override admin password (must be >= 8 chars)
#
# Why this file:
# - One-line start for Win users (Bash-style `VAR=x cmd` doesn't work in PS)
# - Bakes in :3001 / data-dev/ defaults from docs/DEV.md so commands stay short
# - Auto-detects whether `air` is on PATH; falls back to `go run` with a hint
#
# See docs/DEV.md for the full workflow + Troubleshooting (PowerShell vs Bash
# env-var syntax, air PATH setup, password length rule, etc).

param(
  [switch]$NoAir,
  [int]$Port = 3001,
  [string]$Pw = 'devdev99',
  [string]$DataDir = './data-dev'
)

$ErrorActionPreference = 'Stop'

if ($Pw.Length -lt 8) {
  Write-Host "ERROR: -Pw must be >= 8 characters (Phase 3d-1 strong password rule)." -ForegroundColor Red
  exit 1
}

$env:MOON_PORT = "$Port"
$env:MOON_ADMIN_PASSWORD = $Pw
$env:MOON_DATA_DIR = $DataDir

Write-Host ""
Write-Host "Moon Panel dev backend" -ForegroundColor Cyan
Write-Host ("  Port: {0}" -f $Port)
Write-Host ("  Data: {0}" -f $DataDir)
Write-Host ("  Admin: admin / {0}" -f $Pw)
Write-Host ""

# Resolve `go` and `air` binaries. PATH-first, then common install locations
# (helps when user just installed Go but the existing PS session inherited an
# older PATH — common Win pitfall, no terminal restart needed this way).
function Resolve-Tool($name, [string[]]$fallbacks) {
  $cmd = Get-Command $name -ErrorAction SilentlyContinue
  if ($cmd) { return $cmd.Source }
  foreach ($p in $fallbacks) {
    if (Test-Path $p) { return $p }
  }
  return $null
}

$goExe = Resolve-Tool 'go' @(
  'C:\Program Files\Go\bin\go.exe',
  "$env:USERPROFILE\go\bin\go.exe",
  'C:\Go\bin\go.exe'
)
if (-not $goExe) {
  Write-Host "ERROR: 'go' not found on PATH or in common install locations." -ForegroundColor Red
  Write-Host "Install Go 1.23+ from https://go.dev/dl/ then reopen the terminal." -ForegroundColor Red
  exit 1
}

$airExe = Resolve-Tool 'air' @(
  "$env:USERPROFILE\go\bin\air.exe"
)

# Auto-detect air unless -NoAir is set.
$useAir = (-not $NoAir) -and ($null -ne $airExe)
if ((-not $NoAir) -and ($null -eq $airExe)) {
  Write-Host "air not found on PATH or in $env:USERPROFILE\go\bin — falling back to 'go run' (no hot reload)." -ForegroundColor Yellow
  Write-Host "Install once: $goExe install github.com/air-verse/air@latest" -ForegroundColor Yellow
  Write-Host ""
}

if ($useAir) {
  Write-Host "Starting via air (hot reload)..." -ForegroundColor Green
  & $airExe
} else {
  Write-Host "Starting via go run (manual restart on .go changes)..." -ForegroundColor Green
  & $goExe run ./cmd/server
}
