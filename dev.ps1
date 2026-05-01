$env:MOON_PORT = "3001"
$env:MOON_ADMIN_PASSWORD = "dev"
$env:MOON_DATA_DIR = "./data-dev"
$env:Path += ";$env:USERPROFILE\go\bin"

Write-Host "Starting Moon Panel dev backend on :3001..." -ForegroundColor Green
air