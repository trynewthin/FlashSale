# reset-env.ps1 - 完全清理并重新启动整个开发环境 (PowerShell 版)

$RepoRoot = Resolve-Path "$PSScriptRoot\.."
$ComposeFile = "$RepoRoot\deploy\compose\docker-compose.app.yml"

Write-Host "⚠️  This will DELETE all data and restart everything. Starting in 3 seconds..." -ForegroundColor Yellow
Start-Sleep -Seconds 3

Write-Host ">>> [1/4] Stopping and removing all containers/volumes..." -ForegroundColor Cyan
docker compose -f "$ComposeFile" down -v

Write-Host ">>> [2/4] Rebuilding all backend images..." -ForegroundColor Cyan
docker compose -f "$ComposeFile" --profile build build backend-image
docker compose -f "$ComposeFile" build --no-cache

Write-Host ">>> [3/4] Starting environment (databases, infra)..." -ForegroundColor Cyan
docker compose -f "$ComposeFile" up -d mysql redis kafka etcd
Write-Host "Waiting for infra health..."
Start-Sleep -Seconds 10

Write-Host ">>> [4/4] Starting all services and ops-control..." -ForegroundColor Cyan
docker compose -f "$ComposeFile" up -d
docker compose -f "$ComposeFile" up -d ops-control

Write-Host "✅ Environment Reset Complete." -ForegroundColor Green
docker compose -f "$ComposeFile" ps
