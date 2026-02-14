param(
    [bool]$AutoLoadDevEnv = $true,
    [string]$EnvFile = "configs/local/dev.env",
    [switch]$KillExisting = $true,
    [switch]$BootstrapAdmin = $true,
    [string]$BootstrapUsername = "admin_root",
    [string]$BootstrapPassword = "Admin12345",
    [string]$BootstrapDisplayName = "Super Admin",
    [int]$PortReadyTimeoutSec = 90
)

$ErrorActionPreference = "Stop"
$repoRoot = (Resolve-Path (Join-Path $PSScriptRoot "..\..\..\")).Path
$logDir = Join-Path $repoRoot ".memory/runlogs/services"
New-Item -ItemType Directory -Force -Path $logDir | Out-Null

. (Join-Path $PSScriptRoot "..\common.ps1")
if ($AutoLoadDevEnv) {
    $devEnvPath = Join-Path $repoRoot $EnvFile
    if (Test-Path $devEnvPath) {
        Load-DevEnv -Path $devEnvPath
    }
}

if ($BootstrapAdmin) {
    $env:ADMIN_BOOTSTRAP_ENABLED = "true"
    $env:ADMIN_BOOTSTRAP_USERNAME = $BootstrapUsername
    $env:ADMIN_BOOTSTRAP_PASSWORD = $BootstrapPassword
    $env:ADMIN_BOOTSTRAP_DISPLAY_NAME = $BootstrapDisplayName
}

function Stop-PortProcess {
    param([int]$Port)
    $listen = Get-NetTCPConnection -State Listen -LocalPort $Port -ErrorAction SilentlyContinue | Select-Object -First 1
    if ($listen) {
        Stop-Process -Id $listen.OwningProcess -Force -ErrorAction SilentlyContinue
        Start-Sleep -Milliseconds 500
    }
}

function Wait-PortReady {
    param(
        [int]$Port,
        [int]$TimeoutSec = 90
    )
    $deadline = (Get-Date).AddSeconds($TimeoutSec)
    while ((Get-Date) -lt $deadline) {
        if (Get-NetTCPConnection -State Listen -LocalPort $Port -ErrorAction SilentlyContinue) {
            return $true
        }
        Start-Sleep -Milliseconds 400
    }
    return $false
}

function Get-ListenPid {
    param([int]$Port)
    $listen = Get-NetTCPConnection -State Listen -LocalPort $Port -ErrorAction SilentlyContinue | Select-Object -First 1
    if ($listen) {
        return [int]$listen.OwningProcess
    }
    return 0
}

function Start-ServiceProcess {
    param(
        [string]$Name,
        [string[]]$ArgumentList
    )
    $stdoutPath = Join-Path $logDir "$Name.stdout.log"
    $stderrPath = Join-Path $logDir "$Name.stderr.log"
    if (Test-Path $stdoutPath) { Remove-Item $stdoutPath -Force }
    if (Test-Path $stderrPath) { Remove-Item $stderrPath -Force }

    $proc = Start-Process -FilePath "go" `
        -ArgumentList $ArgumentList `
        -WorkingDirectory $repoRoot `
        -PassThru `
        -WindowStyle Hidden `
        -RedirectStandardOutput $stdoutPath `
        -RedirectStandardError $stderrPath
    return $proc
}

function Assert-Healthz {
    param([string]$Url)
    $resp = Invoke-WebRequest -UseBasicParsing -TimeoutSec 5 $Url
    if ($resp.StatusCode -ne 200) {
        throw "health check failed: $Url => $($resp.StatusCode)"
    }
}

$serviceDefs = @(
    @{ Name = "user-rpc"; Port = 8081; Args = @("run", "apps/user/rpc/user.go", "-f", "apps/user/rpc/etc/user.yaml") },
    @{ Name = "product-rpc"; Port = 8084; Args = @("run", "apps/product/rpc/product.go", "-f", "apps/product/rpc/etc/product.yaml") },
    @{ Name = "admin-rpc"; Port = 8087; Args = @("run", "apps/admin/rpc/admin.go", "-f", "apps/admin/rpc/etc/admin.yaml") },
    @{ Name = "order-rpc"; Port = 8085; Args = @("run", "apps/order/rpc/order.go", "-f", "apps/order/rpc/etc/order.yaml") },
    @{ Name = "seckill-rpc"; Port = 8086; Args = @("run", "apps/seckill/rpc/seckill.go", "-f", "apps/seckill/rpc/etc/seckill.yaml") },
    @{ Name = "user-gateway"; Port = 8082; Args = @("run", "apps/gateway/user/main.go", "-f", "apps/gateway/user/etc/user-gateway.yaml") },
    @{ Name = "admin-gateway"; Port = 8083; Args = @("run", "apps/gateway/admin/main.go", "-f", "apps/gateway/admin/etc/admin-gateway.yaml") }
)

if ($KillExisting) {
    foreach ($svc in $serviceDefs) {
        Stop-PortProcess -Port $svc.Port
    }
}

$started = @()
foreach ($svc in $serviceDefs) {
    Write-Host "[start-backend] starting $($svc.Name) on :$($svc.Port)"
    $proc = Start-ServiceProcess -Name $svc.Name -ArgumentList $svc.Args
    if (-not (Wait-PortReady -Port $svc.Port -TimeoutSec $PortReadyTimeoutSec)) {
        throw "$($svc.Name) not ready on port $($svc.Port). check logs in $logDir"
    }
    $listenPid = Get-ListenPid -Port $svc.Port
    $started += [PSCustomObject]@{
        Name        = $svc.Name
        Port        = $svc.Port
        ListenPID   = $listenPid
        LauncherPID = $proc.Id
    }
}

Assert-Healthz -Url "http://127.0.0.1:8082/healthz"
Assert-Healthz -Url "http://127.0.0.1:8083/healthz"

$pidFile = Join-Path $logDir "backend.pids.json"
$started | ConvertTo-Json -Depth 3 | Set-Content -Path $pidFile -Encoding utf8

Write-Host "[start-backend] all services ready"
$started | ForEach-Object { Write-Host ("  - {0} :{1} listen_pid={2} launcher_pid={3}" -f $_.Name, $_.Port, $_.ListenPID, $_.LauncherPID) }
Write-Host "[start-backend] health: user-gateway/admin-gateway = 200"
Write-Host "[start-backend] pid file: $pidFile"

