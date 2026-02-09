param(
    [string]$BaseUrl = "http://127.0.0.1:8083",
    [string]$AdminToken = "",
    [string]$AdminTokenFile = ".memory/runlogs/admin.token.txt",
    [string]$AdminUsername = "",
    [string]$AdminPassword = "",
    [long]$ProductId = 0,
    [string]$ProductIdFile = ".memory/runlogs/perf.product_id.txt",
    [int]$DurationMinutes = 30,
    [long]$SeckillPriceCent = 9900,
    [long]$ReservedStockTotal = 5000,
    [long]$UserLimitQty = 0,
    [long]$MaxQtyPerOrder = 2,
    [string]$OutputDir = ".memory/runlogs"
)

$ErrorActionPreference = "Stop"

function Resolve-Token {
    param([string]$Token, [string]$TokenFile, [string]$Username, [string]$Password, [string]$Base)
    $usernameTrim = $Username.Trim()
    $passwordTrim = $Password.Trim()
    if (-not [string]::IsNullOrWhiteSpace($usernameTrim) -and -not [string]::IsNullOrWhiteSpace($passwordTrim)) {
        $loginUrl = "$($Base.TrimEnd('/'))/api/v1/admin/auth/login"
        $loginBody = @{ username = $usernameTrim; password = $passwordTrim } | ConvertTo-Json -Compress
        $loginResp = Invoke-RestMethod -Method Post -Uri $loginUrl -ContentType "application/json" -Body $loginBody
        if ($null -eq $loginResp -or $loginResp.code -ne "OK" -or [string]::IsNullOrWhiteSpace($loginResp.data.access_token)) {
            throw "管理员登录失败: $($loginResp | ConvertTo-Json -Depth 10 -Compress)"
        }
        return $loginResp.data.access_token.Trim()
    }

    $v = $Token.Trim()
    if (-not [string]::IsNullOrWhiteSpace($v)) {
        return $v
    }
    if (-not (Test-Path $TokenFile)) {
        throw "管理员 token 文件不存在: $TokenFile"
    }
    $raw = (Get-Content -Raw $TokenFile).Trim()
    if ([string]::IsNullOrWhiteSpace($raw)) {
        throw "管理员 token 为空: $TokenFile"
    }
    return $raw
}

function Resolve-ProductId {
    param([long]$Id, [string]$IdFile)
    if ($Id -gt 0) {
        return $Id
    }
    if (-not (Test-Path $IdFile)) {
        throw "商品 ID 文件不存在: $IdFile"
    }
    $raw = (Get-Content -Raw $IdFile).Trim()
    [long]$parsed = 0
    if (-not [long]::TryParse($raw, [ref]$parsed) -or $parsed -le 0) {
        throw "商品 ID 非法: $raw"
    }
    return $parsed
}

function Invoke-Api {
    param(
        [string]$Method,
        [string]$Url,
        [string]$Token,
        [object]$Body
    )
    $headers = @{
        "Authorization" = "Bearer $Token"
        "Content-Type"  = "application/json"
    }
    $payload = if ($null -eq $Body) { "" } else { ($Body | ConvertTo-Json -Depth 10 -Compress) }
    $resp = Invoke-RestMethod -Method $Method -Uri $Url -Headers $headers -Body $payload
    if ($null -eq $resp -or $resp.code -ne "OK") {
        throw "请求失败: $Method $Url => $($resp | ConvertTo-Json -Depth 10 -Compress)"
    }
    return $resp.data
}

$token = Resolve-Token -Token $AdminToken -TokenFile $AdminTokenFile -Username $AdminUsername -Password $AdminPassword -Base $BaseUrl
$productID = Resolve-ProductId -Id $ProductId -IdFile $ProductIdFile

$nowUnix = [DateTimeOffset]::UtcNow.ToUnixTimeSeconds()
$startAt = $nowUnix - 60
$endAt = $nowUnix + ($DurationMinutes * 60)
if ($endAt -le $startAt) {
    throw "活动结束时间必须晚于开始时间"
}

$titleSuffix = [DateTimeOffset]::UtcNow.ToUnixTimeMilliseconds()
$createActivityUrl = "$($BaseUrl.TrimEnd('/'))/api/v1/admin/seckill/activities"
$activity = Invoke-Api -Method "POST" -Url $createActivityUrl -Token $token -Body @{
    title             = "压测活动-$titleSuffix"
    description       = "seckill perf prepared activity"
    style_config_json = "{}"
    start_at_unix     = $startAt
    end_at_unix       = $endAt
}

[long]$activityID = $activity.activity.activity_id
if ($activityID -le 0) {
    throw "创建活动失败，activity_id 非法"
}

$createItemUrl = "$($BaseUrl.TrimEnd('/'))/api/v1/admin/seckill/activities/$activityID/items"
$item = Invoke-Api -Method "POST" -Url $createItemUrl -Token $token -Body @{
    product_id           = $productID
    seckill_price_cent   = $SeckillPriceCent
    reserved_stock_total = $ReservedStockTotal
    user_limit_mode      = 0
    user_limit_window_sec = 0
    user_limit_qty       = $UserLimitQty
    max_qty_per_order    = $MaxQtyPerOrder
    status               = 1
}

[long]$itemID = $item.item.item_id
if ($itemID -le 0) {
    throw "创建活动商品失败，item_id 非法"
}

$publishUrl = "$($BaseUrl.TrimEnd('/'))/api/v1/admin/seckill/activities/$activityID/publish"
[void](Invoke-Api -Method "POST" -Url $publishUrl -Token $token -Body @{})

if (-not (Test-Path $OutputDir)) {
    New-Item -ItemType Directory -Path $OutputDir | Out-Null
}
Set-Content -Path (Join-Path $OutputDir "perf.activity_id.txt") -Value $activityID
Set-Content -Path (Join-Path $OutputDir "perf.item_id.txt") -Value $itemID
Set-Content -Path (Join-Path $OutputDir "perf.product_id.txt") -Value $productID

Write-Host "[seckill-prepare] activity_id=$activityID item_id=$itemID product_id=$productID"
Write-Host "[seckill-prepare] window=$startAt..$endAt (unix seconds)"
