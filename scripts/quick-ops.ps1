# quick-ops.ps1 - 最快的前端更新方式
# 跳过 Docker 镜像构建，直接本地编译 + docker cp 替换二进制
# 耗时：~10-15s

$ErrorActionPreference = "Stop"
$RepoRoot = Resolve-Path "$PSScriptRoot\.."
$FrontendDir = "$RepoRoot\frontend\ops"
$WebDir = "$RepoRoot\cmd\fs\internal\ops\web"
$Container = "flashsale-app-ops-control"
$TmpBin = "$RepoRoot\.tmp-fs-bin"

# ─── [1/4] 构建前端 ───
Write-Host ">>> [1/4] Building frontend..." -ForegroundColor Cyan
Push-Location $FrontendDir
try {
    bun run build
    if ($LASTEXITCODE -ne 0) { throw "frontend build failed" }
} finally {
    Pop-Location
}

# 同步前端产物到 Go embed 目录
Write-Host ">>> Syncing dist -> ops/web..." -ForegroundColor Yellow
if (Test-Path $WebDir) { Remove-Item -Recurse -Force $WebDir }
Copy-Item -Recurse "$FrontendDir\dist" $WebDir

# ─── [2/4] 本地交叉编译 fs 二进制（Linux/amd64） ───
Write-Host ">>> [2/4] Cross-compiling fs binary (linux/amd64)..." -ForegroundColor Cyan
$env:GOOS = "linux"
$env:GOARCH = "amd64"
$env:CGO_ENABLED = "0"

Push-Location $RepoRoot
try {
    go build -buildvcs=false -trimpath -o $TmpBin ./cmd/fs
    if ($LASTEXITCODE -ne 0) { throw "go build failed" }
} finally {
    # 恢复环境变量
    Remove-Item Env:\GOOS -ErrorAction SilentlyContinue
    Remove-Item Env:\GOARCH -ErrorAction SilentlyContinue
    Remove-Item Env:\CGO_ENABLED -ErrorAction SilentlyContinue
    Pop-Location
}

# ─── [3/4] docker cp 替换容器内二进制 ───
Write-Host ">>> [3/4] Copying binary into container..." -ForegroundColor Cyan
docker cp $TmpBin "${Container}:/app/bin/fs"
Remove-Item $TmpBin -Force

# ─── [4/4] 重启容器 ───
Write-Host ">>> [4/4] Restarting container..." -ForegroundColor Cyan
docker restart $Container

# 快速健康检查
Write-Host ">>> Waiting for healthcheck..." -ForegroundColor Yellow
for ($i = 1; $i -le 10; $i++) {
    Start-Sleep -Seconds 2
    $Status = docker inspect --format='{{.State.Health.Status}}' $Container 2>$null
    if ($Status -eq "healthy") {
        Write-Host " Ops Control is HEALTHY!" -ForegroundColor Green
        return
    }
}
Write-Host " Health check timeout (may still be starting)" -ForegroundColor Yellow
