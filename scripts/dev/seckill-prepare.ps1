param(
    [string]$BaseUrl = "http://127.0.0.1:8083",
    [bool]$AutoLoadDevEnv = $true,
    [string]$EnvFile = "configs/local/dev.env",
    [string]$AdminToken = "",
    [string]$AdminTokenFile = ".memory/runlogs/admin.token.txt",
    [string]$AdminUsername = $env:FLASHSALE_ADMIN_USERNAME,
    [string]$AdminPassword = $env:FLASHSALE_ADMIN_PASSWORD,
    [long]$ProductId = 0,
    [string]$ProductIdFile = ".memory/runlogs/perf.product_id.txt",
    [int]$DurationMinutes = 30,
    [long]$SeckillPriceCent = 9900,
    [long]$ReservedStockTotal = 5000,
    [switch]$EnsureProductStock = $true,
    [long]$ProductStockTarget = 60000,
    [long]$UserLimitQty = 0,
    [long]$MaxQtyPerOrder = 2,
    [string]$OutputDir = ".memory/runlogs"
)

$ErrorActionPreference = "Stop"

$repoRoot = (Resolve-Path (Join-Path $PSScriptRoot "..\..\")).Path
if ($AutoLoadDevEnv) {
    . (Join-Path $PSScriptRoot "common.ps1")
    $envPath = Join-Path $repoRoot $EnvFile
    if (Test-Path $envPath) {
        Load-DevEnv -Path $envPath
    }
}
if ([string]::IsNullOrWhiteSpace($AdminUsername) -and -not [string]::IsNullOrWhiteSpace($env:FLASHSALE_ADMIN_USERNAME)) {
    $AdminUsername = $env:FLASHSALE_ADMIN_USERNAME
}
if ([string]::IsNullOrWhiteSpace($AdminPassword) -and -not [string]::IsNullOrWhiteSpace($env:FLASHSALE_ADMIN_PASSWORD)) {
    $AdminPassword = $env:FLASHSALE_ADMIN_PASSWORD
}

function Resolve-Token {
    param([string]$Token, [string]$TokenFile, [string]$Username, [string]$Password, [string]$Base)
    $validateToken = {
        param([string]$TokenToVerify)
        if ([string]::IsNullOrWhiteSpace($TokenToVerify)) {
            return $false
        }
        try {
            $headers = @{ Authorization = "Bearer $TokenToVerify" }
            $resp = Invoke-RestMethod -Method Get -Uri "$($Base.TrimEnd('/'))/api/v1/admin/me" -Headers $headers
            return $resp.code -eq "OK"
        } catch {
            return $false
        }
    }

    $usernameTrim = $Username.Trim()
    $passwordTrim = $Password.Trim()
    $canLogin = (-not [string]::IsNullOrWhiteSpace($usernameTrim) -and -not [string]::IsNullOrWhiteSpace($passwordTrim))
    $login = {
        $loginUrl = "$($Base.TrimEnd('/'))/api/v1/admin/auth/login"
        $loginBody = @{ username = $usernameTrim; password = $passwordTrim } | ConvertTo-Json -Compress
        $loginResp = Invoke-RestMethod -Method Post -Uri $loginUrl -ContentType "application/json" -Body $loginBody
        if ($null -eq $loginResp -or $loginResp.code -ne "OK" -or [string]::IsNullOrWhiteSpace($loginResp.data.access_token)) {
            throw "admin login failed: $($loginResp | ConvertTo-Json -Depth 10 -Compress)"
        }
        return $loginResp.data.access_token.Trim()
    }

    $v = $Token.Trim()
    if (-not [string]::IsNullOrWhiteSpace($v)) {
        if ((& $validateToken $v)) {
            return $v
        }
        if ($canLogin) {
            Write-Host "[seckill-prepare] admin token invalid, fallback to username/password login"
            return (& $login)
        }
        throw "admin token invalid, and no username/password provided"
    }
    if (-not (Test-Path $TokenFile)) {
        if ($canLogin) {
            return (& $login)
        }
        throw "admin token file not found: $TokenFile"
    }
    $raw = (Get-Content -Raw $TokenFile).Trim()
    if ([string]::IsNullOrWhiteSpace($raw)) {
        if ($canLogin) {
            return (& $login)
        }
        throw "admin token is empty: $TokenFile"
    }
    if ((& $validateToken $raw)) {
        return $raw
    }
    if ($canLogin) {
        Write-Host "[seckill-prepare] admin token file expired, fallback to username/password login"
        return (& $login)
    }
    throw "admin token from file is invalid: $TokenFile"
}

function Resolve-ProductId {
    param([long]$Id, [string]$IdFile)
    if ($Id -gt 0) {
        return $Id
    }
    if (-not (Test-Path $IdFile)) {
        throw "product id file not found: $IdFile"
    }
    $raw = (Get-Content -Raw $IdFile).Trim()
    [long]$parsed = 0
    if (-not [long]::TryParse($raw, [ref]$parsed) -or $parsed -le 0) {
        throw "invalid product id: $raw"
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
        throw "request failed: $Method $Url => $($resp | ConvertTo-Json -Depth 10 -Compress)"
    }
    return $resp.data
}

function Ensure-ProductStockLevel {
    param(
        [string]$Base,
        [string]$Token,
        [long]$ProductID,
        [long]$MinStock
    )
    if ($MinStock -le 0) {
        return
    }
    $detailUrl = "$($Base.TrimEnd('/'))/api/v1/admin/products/$ProductID"
    $headers = @{
        "Authorization" = "Bearer $Token"
    }
    $detail = Invoke-RestMethod -Method Get -Uri $detailUrl -Headers $headers
    if ($null -eq $detail -or $detail.code -ne "OK" -or $null -eq $detail.data.product) {
        throw "query product failed: $ProductID"
    }
    $product = $detail.data.product
    [long]$currentStock = [long]$product.stock
    if ($currentStock -ge $MinStock) {
        Write-Host "[seckill-prepare] product stock ready: current=$currentStock target=$MinStock"
        return
    }
    $updateUrl = "$($Base.TrimEnd('/'))/api/v1/admin/products/$ProductID"
    [void](Invoke-Api -Method "PATCH" -Url $updateUrl -Token $Token -Body @{
        name        = [string]$product.name
        main_image  = [string]$product.main_image
        description = [string]$product.description
        price_cent  = [long]$product.price_cent
        stock       = $MinStock
        status      = [int]$product.status
    })
    Write-Host "[seckill-prepare] product stock topped up: $currentStock -> $MinStock"
}

$token = Resolve-Token -Token $AdminToken -TokenFile $AdminTokenFile -Username $AdminUsername -Password $AdminPassword -Base $BaseUrl
$productID = Resolve-ProductId -Id $ProductId -IdFile $ProductIdFile
if ($EnsureProductStock) {
    $targetStock = $ProductStockTarget
    $minByReserved = [Math]::Max([long]($ReservedStockTotal * 2), 0)
    if ($targetStock -lt $minByReserved) {
        $targetStock = $minByReserved
    }
    Ensure-ProductStockLevel -Base $BaseUrl -Token $token -ProductID $productID -MinStock $targetStock
}

$nowUnix = [DateTimeOffset]::UtcNow.ToUnixTimeSeconds()
$startAt = $nowUnix - 60
$endAt = $nowUnix + ($DurationMinutes * 60)
if ($endAt -le $startAt) {
    throw "end_at must be later than start_at"
}

$titleSuffix = [DateTimeOffset]::UtcNow.ToUnixTimeMilliseconds()
$createActivityUrl = "$($BaseUrl.TrimEnd('/'))/api/v1/admin/seckill/activities"
$activity = Invoke-Api -Method "POST" -Url $createActivityUrl -Token $token -Body @{
    title             = "perf-activity-$titleSuffix"
    description       = "seckill perf prepared activity"
    style_config_json = "{}"
    start_at_unix     = $startAt
    end_at_unix       = $endAt
}

[long]$activityID = $activity.activity.activity_id
if ($activityID -le 0) {
    throw "create activity failed, invalid activity_id"
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
    throw "create activity item failed, invalid item_id"
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
