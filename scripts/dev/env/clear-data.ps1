param(
    [switch]$Force,
    [switch]$ClearAdmin,
    [switch]$KeepInfraTables,
    [switch]$SkipRedisFlush,
    [bool]$AutoLoadDevEnv = $true,
    [string]$EnvFile = "configs/local/dev.env"
)

$ErrorActionPreference = "Stop"
$repoRoot = (Resolve-Path (Join-Path $PSScriptRoot "..\..\..\")).Path

if (-not $Force) {
    throw "clear-data is destructive, pass -Force explicitly"
}

. (Join-Path $PSScriptRoot "..\common.ps1")
if ($AutoLoadDevEnv) {
    $envPath = Join-Path $repoRoot $EnvFile
    if (Test-Path $envPath) {
        Load-DevEnv -Path $envPath
    }
}

$mysqlUser = if ($env:FLASHSALE_MYSQL_USER) { $env:FLASHSALE_MYSQL_USER } else { $env:FLASH_MYSQL_APP_USER }
$mysqlPass = if ($env:FLASHSALE_MYSQL_PASSWORD) { $env:FLASHSALE_MYSQL_PASSWORD } else { $env:FLASH_MYSQL_APP_PASSWORD }
if ([string]::IsNullOrWhiteSpace($mysqlUser) -or [string]::IsNullOrWhiteSpace($mysqlPass)) {
    throw "mysql user/password not found from env"
}

function Invoke-MySQL {
    param(
        [string]$Database,
        [string]$Sql
    )
    $args = @(
        "exec",
        "flashsale-mysql",
        "mysql",
        ("-u{0}" -f $mysqlUser),
        ("-p{0}" -f $mysqlPass),
        $Database,
        "-e",
        $Sql
    )
    & docker @args | Out-Null
    if ($LASTEXITCODE -ne 0) {
        throw "mysql exec failed on $Database"
    }
}

if (-not $SkipRedisFlush) {
    Write-Host "[clear-data] flush redis db"
    & docker exec flashsale-redis redis-cli FLUSHDB | Out-Null
}

function Add-InfraCleanup {
    param([string[]]$Tables)
    if ($KeepInfraTables) {
        return $Tables
    }
    return ($Tables + @("outbox_events", "idempotency_records"))
}

function Build-TruncateSQL {
    param([string[]]$Tables)
    $sqlLines = @("SET FOREIGN_KEY_CHECKS=0;")
    foreach ($tbl in $Tables) {
        $sqlLines += "TRUNCATE TABLE $tbl;"
    }
    $sqlLines += "SET FOREIGN_KEY_CHECKS=1;"
    return ($sqlLines -join "`n")
}

Write-Host "[clear-data] truncate order/seckill/product/user business data"
$orderTables = Add-InfraCleanup -Tables @("order_events", "orders")
$seckillTables = Add-InfraCleanup -Tables @(
    "seckill_stock_ledger",
    "seckill_traffic_agg_minute",
    "seckill_traffic_raw_events",
    "seckill_order_links",
    "seckill_activity_items",
    "seckill_activities"
)
$productTables = Add-InfraCleanup -Tables @("products")
$userTables = Add-InfraCleanup -Tables @("users")

Invoke-MySQL -Database "flash_order" -Sql (Build-TruncateSQL -Tables $orderTables)
Invoke-MySQL -Database "flash_seckill" -Sql (Build-TruncateSQL -Tables $seckillTables)
Invoke-MySQL -Database "flash_product" -Sql (Build-TruncateSQL -Tables $productTables)
Invoke-MySQL -Database "flash_user" -Sql (Build-TruncateSQL -Tables $userTables)

if ($ClearAdmin) {
    Write-Host "[clear-data] truncate admin data (including admins/roles)"
    $adminTables = @(
        "admin_audit_logs",
        "admin_refresh_tokens",
        "admin_role_bindings",
        "admin_role_domains",
        "admin_roles",
        "admins"
    )
    $adminTables = Add-InfraCleanup -Tables $adminTables
    Invoke-MySQL -Database "flash_admin" -Sql (Build-TruncateSQL -Tables $adminTables)
} else {
    Write-Host "[clear-data] truncate admin volatile data only (keep admins/roles)"
    $adminTables = @("admin_audit_logs", "admin_refresh_tokens")
    if (-not $KeepInfraTables) {
        $adminTables += @("outbox_events", "idempotency_records")
    }
    Invoke-MySQL -Database "flash_admin" -Sql (Build-TruncateSQL -Tables $adminTables)
}

Write-Host "[clear-data] done"
Write-Host ("  - clear_admin: {0}" -f [bool]$ClearAdmin)
Write-Host ("  - keep_infra_tables: {0}" -f [bool]$KeepInfraTables)
Write-Host ("  - skip_redis_flush: {0}" -f [bool]$SkipRedisFlush)

