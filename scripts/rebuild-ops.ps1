# rebuild-ops.ps1 - 构建前端 + 后端并重启 Ops Control 及可观测性服务 (PowerShell 版)

$RepoRoot = Resolve-Path "$PSScriptRoot\.."
$ComposeFile = "$RepoRoot\deploy\compose\docker-compose.app.yml"
$FrontendDir = "$RepoRoot\frontend\ops"
$WebDir = "$RepoRoot\cmd\fs\internal\ops\web"

# ─── [1/5] 构建前端 ───
Write-Host ">>> [1/5] Building frontend..." -ForegroundColor Cyan
Push-Location $FrontendDir
try {
    bun run build
    if ($LASTEXITCODE -ne 0) { throw "frontend build failed" }
} finally {
    Pop-Location
}

# 同步前端产物到 Go embed 目录
Write-Host ">>> Syncing frontend dist -> ops/web..." -ForegroundColor Yellow
if (Test-Path $WebDir) { Remove-Item -Recurse -Force $WebDir }
Copy-Item -Recurse "$FrontendDir\dist" $WebDir

# ─── [2/5] 构建后端基础镜像 ───
Write-Host ">>> [2/5] Building backend base image..." -ForegroundColor Cyan
docker compose -f "$ComposeFile" --profile build build backend-image

# ─── [3/5] 构建 ops-control 镜像 ───
Write-Host ">>> [3/5] Building ops-control image (no-cache)..." -ForegroundColor Cyan
docker compose -f "$ComposeFile" build --no-cache ops-control

# ─── [4/5] 重启 ops-control ───
Write-Host ">>> [4/5] Restarting ops-control service..." -ForegroundColor Cyan
docker compose -f "$ComposeFile" up -d ops-control --force-recreate

# ─── [5/5] 启动可观测性组件 ───
Write-Host ">>> [5/5] Starting observability services (prometheus, grafana, jaeger)..." -ForegroundColor Cyan
docker compose -f "$ComposeFile" --profile observability up -d

Write-Host ">>> Waiting for healthcheck..." -ForegroundColor Yellow
for ($i = 1; $i -le 30; $i++) {
    $Status = docker inspect --format='{{.State.Health.Status}}' flashsale-app-ops-control 2>$null
    if ($Status -eq "healthy") {
        Write-Host "✅ Ops Control is HEALTHY!" -ForegroundColor Green
        # 显示可观测性服务状态
        Write-Host ">>> Observability services:" -ForegroundColor Yellow
        docker ps --filter "name=flashsale-app-prometheus" --filter "name=flashsale-app-grafana" --filter "name=flashsale-app-jaeger" --format "  {{.Names}}: {{.Status}}"
        return
    }
    Write-Host "... current status: $Status (waiting $i/30)"
    Start-Sleep -Seconds 2
}

Write-Host "❌ Ops Control failed to become healthy in time." -ForegroundColor Red
docker logs flashsale-app-ops-control --tail 20
exit 1
