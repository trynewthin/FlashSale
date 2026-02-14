param(
    [switch]$Force,
    [bool]$AutoLoadDevEnv = $true,
    [string]$EnvFile = "configs/local/dev.env",
    [string]$AdminBaseUrl = "http://127.0.0.1:8083",
    [string]$UserBaseUrl = "http://127.0.0.1:8082",
    [string]$AdminUsername = "admin_root",
    [string]$AdminPassword = "Admin12345",
    [string]$AdminDisplayName = "admin root",
    [string]$SeedUserPhone = "13900000001",
    [string]$SeedUserPassword = "abc12345",
    [string]$SeedUserNickname = "seed_user",
    [long]$SeckillReservedStock = 5000,
    [long]$SeckillPriceCent = 9900,
    [long]$SeckillDurationMinutes = 120,
    [string]$OutputDir = ".memory/runlogs"
)

$ErrorActionPreference = "Stop"
$repoRoot = (Resolve-Path (Join-Path $PSScriptRoot "..\..\..\")).Path

if (-not $Force) {
    throw "overwrite mode is destructive, pass -Force explicitly"
}

. (Join-Path $PSScriptRoot "..\common.ps1")
if ($AutoLoadDevEnv) {
    $envPath = Join-Path $repoRoot $EnvFile
    if (Test-Path $envPath) {
        Load-DevEnv -Path $envPath
    }
}

$outputAbs = Join-Path $repoRoot $OutputDir
if (-not (Test-Path $outputAbs)) {
    New-Item -ItemType Directory -Path $outputAbs -Force | Out-Null
}

$mysqlUser = if ($env:FLASHSALE_MYSQL_USER) { $env:FLASHSALE_MYSQL_USER } else { $env:FLASH_MYSQL_APP_USER }
$mysqlPass = if ($env:FLASHSALE_MYSQL_PASSWORD) { $env:FLASHSALE_MYSQL_PASSWORD } else { $env:FLASH_MYSQL_APP_PASSWORD }
if ([string]::IsNullOrWhiteSpace($mysqlUser) -or [string]::IsNullOrWhiteSpace($mysqlPass)) {
    throw "mysql user/password not found from env"
}

function Assert-Healthz {
    param([string]$Url)
    $resp = Invoke-WebRequest -UseBasicParsing -TimeoutSec 5 "$($Url.TrimEnd('/'))/healthz"
    if ($resp.StatusCode -ne 200) {
        throw "health check failed: $Url/healthz => $($resp.StatusCode)"
    }
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

function New-BcryptHash {
    param([string]$Plain)
    $tmp = Join-Path $env:TEMP ("flashsale_seed_hash_{0}.go" -f ([guid]::NewGuid().ToString("N")))
    @'
package main
import (
  "fmt"
  "golang.org/x/crypto/bcrypt"
)
func main() {
  h, err := bcrypt.GenerateFromPassword([]byte("REPLACE_PASSWORD"), bcrypt.DefaultCost)
  if err != nil { panic(err) }
  fmt.Print(string(h))
}
'@ | Set-Content -Path $tmp -Encoding utf8
    try {
        (Get-Content $tmp -Raw).Replace("REPLACE_PASSWORD", $Plain) | Set-Content -Path $tmp -Encoding utf8
        $hash = (& go run $tmp)
        if ($LASTEXITCODE -ne 0 -or [string]::IsNullOrWhiteSpace($hash)) {
            throw "generate bcrypt hash failed"
        }
        return $hash.Trim()
    } finally {
        Remove-Item $tmp -Force -ErrorAction SilentlyContinue
    }
}

function Invoke-Api {
    param(
        [string]$Method,
        [string]$Url,
        [string]$Token,
        [object]$Body
    )
    $headers = @{}
    if (-not [string]::IsNullOrWhiteSpace($Token)) {
        $headers["Authorization"] = "Bearer $Token"
    }

    $jsonBody = $null
    if ($null -ne $Body) {
        $headers["Content-Type"] = "application/json"
        $jsonBody = ($Body | ConvertTo-Json -Depth 10 -Compress)
    }

    try {
        if ($null -eq $jsonBody) {
            $resp = Invoke-RestMethod -Method $Method -Uri $Url -Headers $headers
        } else {
            $resp = Invoke-RestMethod -Method $Method -Uri $Url -Headers $headers -Body $jsonBody
        }
    } catch {
        if ($_.ErrorDetails.Message) {
            throw "request failed: $Method $Url => $($_.ErrorDetails.Message)"
        }
        throw
    }

    if ($null -eq $resp -or $resp.code -ne "OK") {
        throw "request failed: $Method $Url => $($resp | ConvertTo-Json -Depth 10 -Compress)"
    }
    return $resp.data
}

Write-Host "[seed-overwrite] check gateway health"
Assert-Healthz -Url $AdminBaseUrl
Assert-Healthz -Url $UserBaseUrl

Write-Host "[seed-overwrite] flush redis"
& docker exec flashsale-redis redis-cli FLUSHDB | Out-Null

Write-Host "[seed-overwrite] truncate tables (overwrite mode)"
Invoke-MySQL -Database "flash_order" -Sql @'
SET FOREIGN_KEY_CHECKS=0;
TRUNCATE TABLE order_events;
TRUNCATE TABLE orders;
TRUNCATE TABLE outbox_events;
TRUNCATE TABLE idempotency_records;
SET FOREIGN_KEY_CHECKS=1;
'@

Invoke-MySQL -Database "flash_seckill" -Sql @'
SET FOREIGN_KEY_CHECKS=0;
TRUNCATE TABLE seckill_stock_ledger;
TRUNCATE TABLE seckill_traffic_agg_minute;
TRUNCATE TABLE seckill_traffic_raw_events;
TRUNCATE TABLE seckill_order_links;
TRUNCATE TABLE seckill_activity_items;
TRUNCATE TABLE seckill_activities;
TRUNCATE TABLE outbox_events;
TRUNCATE TABLE idempotency_records;
SET FOREIGN_KEY_CHECKS=1;
'@

Invoke-MySQL -Database "flash_product" -Sql @'
SET FOREIGN_KEY_CHECKS=0;
TRUNCATE TABLE products;
TRUNCATE TABLE outbox_events;
TRUNCATE TABLE idempotency_records;
SET FOREIGN_KEY_CHECKS=1;
'@

Invoke-MySQL -Database "flash_user" -Sql @'
SET FOREIGN_KEY_CHECKS=0;
TRUNCATE TABLE users;
TRUNCATE TABLE outbox_events;
TRUNCATE TABLE idempotency_records;
SET FOREIGN_KEY_CHECKS=1;
'@

Invoke-MySQL -Database "flash_admin" -Sql @'
SET FOREIGN_KEY_CHECKS=0;
TRUNCATE TABLE admin_audit_logs;
TRUNCATE TABLE admin_refresh_tokens;
TRUNCATE TABLE admin_role_bindings;
TRUNCATE TABLE admin_role_domains;
TRUNCATE TABLE admin_roles;
TRUNCATE TABLE admins;
TRUNCATE TABLE outbox_events;
TRUNCATE TABLE idempotency_records;
SET FOREIGN_KEY_CHECKS=1;
'@

$adminPasswordHash = New-BcryptHash -Plain $AdminPassword
$adminID = 9000000000000000010
$roleID = 9000000000000000020
$now = (Get-Date).ToString("yyyy-MM-dd HH:mm:ss")

Write-Host "[seed-overwrite] seed admin root"
$adminSeedSql = @"
INSERT INTO admins (id, username, display_name, password_hash, status, data_scope, is_super_admin, created_at, updated_at)
VALUES ($adminID, '$AdminUsername', '$AdminDisplayName', '$adminPasswordHash', 1, 'all', 1, '$now', '$now');
INSERT INTO admin_roles (id, role_code, role_name, status, is_system, created_at, updated_at)
VALUES ($roleID, 'super_admin', 'Super Admin', 1, 1, '$now', '$now');
INSERT INTO admin_role_bindings (id, admin_id, role_id, created_at)
VALUES (9000000000000000030, $adminID, $roleID, '$now');
INSERT INTO admin_role_domains (id, role_id, domain_code, created_at) VALUES (9000000000000000101, $roleID, 'operations', '$now');
INSERT INTO admin_role_domains (id, role_id, domain_code, created_at) VALUES (9000000000000000102, $roleID, 'user_management', '$now');
INSERT INTO admin_role_domains (id, role_id, domain_code, created_at) VALUES (9000000000000000103, $roleID, 'product_management', '$now');
INSERT INTO admin_role_domains (id, role_id, domain_code, created_at) VALUES (9000000000000000104, $roleID, 'order_management', '$now');
INSERT INTO admin_role_domains (id, role_id, domain_code, created_at) VALUES (9000000000000000105, $roleID, 'order_review_management', '$now');
INSERT INTO admin_role_domains (id, role_id, domain_code, created_at) VALUES (9000000000000000106, $roleID, 'seckill_management', '$now');
INSERT INTO admin_role_domains (id, role_id, domain_code, created_at) VALUES (9000000000000000107, $roleID, 'admin_management', '$now');
"@
Invoke-MySQL -Database "flash_admin" -Sql $adminSeedSql

Write-Host "[seed-overwrite] login admin"
$adminLogin = Invoke-Api -Method "POST" -Url "$($AdminBaseUrl.TrimEnd('/'))/api/v1/admin/auth/login" -Token "" -Body @{
    username = $AdminUsername
    password = $AdminPassword
}
$adminToken = [string]$adminLogin.access_token

Write-Host "[seed-overwrite] seed user"
$null = Invoke-Api -Method "POST" -Url "$($UserBaseUrl.TrimEnd('/'))/api/v1/user/register" -Token "" -Body @{
    phone    = $SeedUserPhone
    password = $SeedUserPassword
    nickname = $SeedUserNickname
}
$userLogin = Invoke-Api -Method "POST" -Url "$($UserBaseUrl.TrimEnd('/'))/api/v1/user/login" -Token "" -Body @{
    phone    = $SeedUserPhone
    password = $SeedUserPassword
}
$userToken = [string]$userLogin.access_token

Write-Host "[seed-overwrite] seed products"
$productDefs = @(
    @{ name = "seed-product-a"; main_image = "https://example.com/seed-a.png"; description = "overwrite seed product a"; price_cent = 19900; stock = 80000; status = 1 },
    @{ name = "seed-product-b"; main_image = "https://example.com/seed-b.png"; description = "overwrite seed product b"; price_cent = 25900; stock = 60000; status = 1 },
    @{ name = "seed-product-c"; main_image = "https://example.com/seed-c.png"; description = "overwrite seed product c"; price_cent = 9900; stock = 50000; status = 1 }
)
$products = @()
foreach ($item in $productDefs) {
    $created = Invoke-Api -Method "POST" -Url "$($AdminBaseUrl.TrimEnd('/'))/api/v1/admin/products" -Token $adminToken -Body $item
    $products += $created.product
}

$firstProductId = [string]$products[0].product_id
$firstProductPrice = [long]$products[0].price_cent

Write-Host "[seed-overwrite] seed seckill activity"
$nowUnix = [DateTimeOffset]::UtcNow.ToUnixTimeSeconds()
$activityResp = Invoke-Api -Method "POST" -Url "$($AdminBaseUrl.TrimEnd('/'))/api/v1/admin/seckill/activities" -Token $adminToken -Body @{
    title             = "seed-activity-$nowUnix"
    description       = "overwrite seckill activity"
    style_config_json = "{}"
    start_at_unix     = $nowUnix - 60
    end_at_unix       = $nowUnix + ($SeckillDurationMinutes * 60)
}
$activityId = [string]$activityResp.activity.activity_id

$itemResp = Invoke-Api -Method "POST" -Url "$($AdminBaseUrl.TrimEnd('/'))/api/v1/admin/seckill/activities/$activityId/items" -Token $adminToken -Body @{
    product_id            = $firstProductId
    seckill_price_cent    = $SeckillPriceCent
    reserved_stock_total  = $SeckillReservedStock
    user_limit_mode       = 0
    user_limit_window_sec = 0
    user_limit_qty        = 0
    max_qty_per_order     = 2
    status                = 1
}
$activityItemId = [string]$itemResp.item.item_id

$null = Invoke-Api -Method "POST" -Url "$($AdminBaseUrl.TrimEnd('/'))/api/v1/admin/seckill/activities/$activityId/publish" -Token $adminToken -Body @{}

Write-Host "[seed-overwrite] seed example orders"
$normalOrder = Invoke-Api -Method "POST" -Url "$($UserBaseUrl.TrimEnd('/'))/api/v1/orders" -Token $userToken -Body @{
    product_id   = $firstProductId
    order_source = 0
}
$seckillOrder = Invoke-Api -Method "POST" -Url "$($UserBaseUrl.TrimEnd('/'))/api/v1/seckill/activities/$activityId/purchase" -Token $userToken -Body @{
    activity_item_id = $activityItemId
    quantity         = 1
    idempotency_key  = ([guid]::NewGuid().ToString("N"))
}

$result = [ordered]@{
    generated_at_unix = [DateTimeOffset]::UtcNow.ToUnixTimeSeconds()
    admin = @{
        username = $AdminUsername
        password = $AdminPassword
        admin_id = $adminLogin.admin.admin_id
    }
    user = @{
        phone    = $SeedUserPhone
        password = $SeedUserPassword
        user_id  = $userLogin.user_id
    }
    products = $products
    seckill = @{
        activity_id      = $activityId
        activity_item_id = $activityItemId
    }
    orders = @{
        normal_order_id  = $normalOrder.order.order_id
        seckill_order_id = $seckillOrder.order_id
    }
}

$resultPath = Join-Path $outputAbs "seed-overwrite.result.json"
$adminTokenPath = Join-Path $outputAbs "admin.token.txt"
$userTokenPath = Join-Path $outputAbs "user.token.txt"
$productPath = Join-Path $outputAbs "perf.product_id.txt"
$activityPath = Join-Path $outputAbs "perf.activity_id.txt"
$itemPath = Join-Path $outputAbs "perf.item_id.txt"

$result | ConvertTo-Json -Depth 10 | Set-Content -Path $resultPath -Encoding utf8
Set-Content -Path $adminTokenPath -Value $adminToken
Set-Content -Path $userTokenPath -Value $userToken
Set-Content -Path $productPath -Value $firstProductId
Set-Content -Path $activityPath -Value $activityId
Set-Content -Path $itemPath -Value $activityItemId

Write-Host "[seed-overwrite] done"
Write-Host "  - result: $resultPath"
Write-Host "  - product_id: $firstProductId"
Write-Host "  - activity_id: $activityId"
Write-Host "  - activity_item_id: $activityItemId"
Write-Host ("  - price baseline: {0} -> seckill {1}" -f $firstProductPrice, $SeckillPriceCent)

