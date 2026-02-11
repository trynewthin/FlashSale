param(
    [switch]$RestartUserGateway = $false,
    [switch]$RestartOrderRPC = $true,
    [switch]$RestartSeckillRPC = $true,
    [string]$EnvFile = "configs/local/dev.env",
    [int]$StartupWaitSeconds = 6,
    [bool]$ForceIPv4Loopback = $true,
    [int]$MySQLMaxOpenConns = 32,
    [int]$MySQLMaxIdleConns = 8,
    [int]$SeckillReservePurchaseTimeoutMs = 5000
)

$ErrorActionPreference = "Stop"
$repoRoot = (Resolve-Path (Join-Path $PSScriptRoot "..\..\")).Path
$envPath = Join-Path $repoRoot $EnvFile

. (Join-Path $PSScriptRoot "common.ps1")
Load-DevEnv -Path $envPath

function Write-Info {
    param([string]$Msg)
    Write-Host "[restart-seckill-runtime] $Msg"
}

function Stop-ListenProcessByPort {
    param([int]$Port)
    $listen = Get-NetTCPConnection -State Listen -LocalPort $Port -ErrorAction SilentlyContinue | Select-Object -First 1
    if ($null -eq $listen) {
        return
    }
    $procId = [int]$listen.OwningProcess
    $proc = Get-Process -Id $procId -ErrorAction SilentlyContinue
    if ($null -eq $proc) {
        return
    }
    Write-Info "stop port=$Port pid=$procId name=$($proc.ProcessName)"
    Stop-Process -Id $procId -Force
    Start-Sleep -Milliseconds 500
}

function Start-GoService {
    param(
        [string]$Name,
        [string[]]$CmdArgs
    )
    $logDir = Join-Path $repoRoot ".memory/runlogs"
    New-Item -ItemType Directory -Force -Path $logDir | Out-Null
    $stdoutFile = Join-Path $logDir "$Name.stdout.log"
    $stderrFile = Join-Path $logDir "$Name.stderr.log"
    Write-Info "start ${Name}: go $($CmdArgs -join ' ')"
    $p = Start-Process -FilePath "go" `
        -ArgumentList $CmdArgs `
        -WorkingDirectory $repoRoot `
        -PassThru `
        -WindowStyle Hidden `
        -RedirectStandardOutput $stdoutFile `
        -RedirectStandardError $stderrFile
    return $p
}

function Wait-PortListening {
    param(
        [int]$Port,
        [int]$TimeoutSeconds = 45
    )
    $deadline = (Get-Date).AddSeconds($TimeoutSeconds)
    while ((Get-Date) -lt $deadline) {
        $listen = Get-NetTCPConnection -State Listen -LocalPort $Port -ErrorAction SilentlyContinue
        if ($null -ne $listen) {
            return
        }
        Start-Sleep -Milliseconds 500
    }
    throw "port not listening: $Port"
}

function Assert-HttpHealthy {
    param([string]$BaseUrl)
    $resp = Invoke-WebRequest -UseBasicParsing -TimeoutSec 2 "$BaseUrl/healthz"
    if ($resp.StatusCode -ne 200) {
        throw "health check failed: $BaseUrl => $($resp.StatusCode)"
    }
}

$services = @()
if ($RestartUserGateway) {
    $services += [pscustomobject]@{
        Name = "user-gateway"
        Port = 8082
        StartArgs = @("run", "apps/gateway/user/main.go", "-f", "apps/gateway/user/etc/user-gateway.yaml")
        NeedHttpHealth = $true
        HealthBase = "http://127.0.0.1:8082"
    }
}
if ($RestartOrderRPC) {
    $services += [pscustomobject]@{
        Name = "order-rpc"
        Port = 8085
        StartArgs = @("run", "apps/order/rpc/order.go", "-f", "apps/order/rpc/etc/order.yaml")
        NeedHttpHealth = $false
        HealthBase = ""
    }
}
if ($RestartSeckillRPC) {
    $services += [pscustomobject]@{
        Name = "seckill-rpc"
        Port = 8086
        StartArgs = @("run", "apps/seckill/rpc/seckill.go", "-f", "apps/seckill/rpc/etc/seckill.yaml")
        NeedHttpHealth = $false
        HealthBase = ""
    }
}

if ($services.Count -eq 0) {
    throw "no service selected"
}

Write-Info "load env: $envPath"
Write-Info "mysql=$($env:FLASHSALE_MYSQL_HOST):$($env:FLASHSALE_MYSQL_PORT) redis=$($env:FLASHSALE_REDIS_ADDR) kafka=$($env:FLASHSALE_KAFKA_BROKERS)"

if ($ForceIPv4Loopback) {
    if ($env:FLASHSALE_MYSQL_HOST -eq "localhost") {
        $env:FLASHSALE_MYSQL_HOST = "127.0.0.1"
    }
    if ($env:FLASHSALE_REDIS_ADDR -like "localhost:*") {
        $env:FLASHSALE_REDIS_ADDR = $env:FLASHSALE_REDIS_ADDR -replace "^localhost:", "127.0.0.1:"
    }
    if (-not [string]::IsNullOrWhiteSpace($env:FLASHSALE_KAFKA_BROKERS)) {
        $brokers = $env:FLASHSALE_KAFKA_BROKERS.Split(",") | ForEach-Object {
            $b = $_.Trim()
            if ($b -like "localhost:*") {
                $b -replace "^localhost:", "127.0.0.1:"
            } else {
                $b
            }
        }
        $env:FLASHSALE_KAFKA_BROKERS = ($brokers -join ",")
    }
}
if ($MySQLMaxOpenConns -gt 0) {
    $env:FLASHSALE_MYSQL_MAX_OPEN_CONNS = "$MySQLMaxOpenConns"
}
if ($MySQLMaxIdleConns -ge 0) {
    $env:FLASHSALE_MYSQL_MAX_IDLE_CONNS = "$MySQLMaxIdleConns"
}
if ($SeckillReservePurchaseTimeoutMs -gt 0) {
    $env:FLASHSALE_SECKILL_RESERVE_PURCHASE_TIMEOUT_MS = "$SeckillReservePurchaseTimeoutMs"
}
Write-Info "runtime overrides: mysql_host=$($env:FLASHSALE_MYSQL_HOST) redis_addr=$($env:FLASHSALE_REDIS_ADDR) kafka_brokers=$($env:FLASHSALE_KAFKA_BROKERS)"
Write-Info "runtime overrides: mysql_max_open_conns=$($env:FLASHSALE_MYSQL_MAX_OPEN_CONNS) mysql_max_idle_conns=$($env:FLASHSALE_MYSQL_MAX_IDLE_CONNS)"
if ($SeckillReservePurchaseTimeoutMs -gt 0) {
    Write-Info "runtime overrides: seckill_reserve_purchase_timeout_ms=$($env:FLASHSALE_SECKILL_RESERVE_PURCHASE_TIMEOUT_MS)"
}

foreach ($svc in $services) {
    Stop-ListenProcessByPort -Port $svc.Port
}

$started = @()
foreach ($svc in $services) {
    $started += Start-GoService -Name $svc.Name -CmdArgs $svc.StartArgs
}

Start-Sleep -Seconds $StartupWaitSeconds

foreach ($svc in $services) {
    Wait-PortListening -Port $svc.Port
    if ($svc.NeedHttpHealth) {
        Assert-HttpHealthy -BaseUrl $svc.HealthBase
    }
    $listen = Get-NetTCPConnection -State Listen -LocalPort $svc.Port -ErrorAction SilentlyContinue | Select-Object -First 1
    if ($null -ne $listen) {
        Write-Info "ready $($svc.Name): port=$($svc.Port) pid=$($listen.OwningProcess)"
    }
}

Write-Info "done"
