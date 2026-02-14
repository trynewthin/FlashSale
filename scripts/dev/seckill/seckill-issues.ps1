param(
    [string]$BaseUrl = "http://127.0.0.1:8082",
    [switch]$Prepare,
    [bool]$AutoLoadDevEnv = $true,
    [string]$EnvFile = "configs/local/dev.env",
    [string]$AdminBaseUrl = "http://127.0.0.1:8083",
    [string]$AdminToken = "",
    [string]$AdminTokenFile = ".memory/runlogs/admin.token.txt",
    [string]$AdminUsername = $env:FLASHSALE_ADMIN_USERNAME,
    [string]$AdminPassword = $env:FLASHSALE_ADMIN_PASSWORD,
    [long]$ProductId = 0,
    [string]$ProductIdFile = ".memory/runlogs/perf.product_id.txt",
    [int]$DurationMinutes = 30,
    [long]$ActivityId,
    [long]$ItemId,
    [string]$Token = "",
    [string]$TokenFile = "",
    [int]$Concurrency = 100,
    [int]$Requests = 1000,
    [int]$ExpectedMaxSuccess = -1,
    [int]$TrackMaxConnsPerHost = 200
)

$ErrorActionPreference = "Stop"

$repoRoot = (Resolve-Path (Join-Path $PSScriptRoot "..\..\..\")).Path
if ($AutoLoadDevEnv) {
    . (Join-Path $PSScriptRoot "..\common.ps1")
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

if ($Prepare) {
    Write-Host "=== 准备压测活动与商品 ==="
    & (Join-Path $repoRoot "scripts/dev/seckill/seckill-prepare.ps1") `
        -BaseUrl $AdminBaseUrl `
        -AdminToken $AdminToken `
        -AdminTokenFile $AdminTokenFile `
        -AdminUsername $AdminUsername `
        -AdminPassword $AdminPassword `
        -ProductId $ProductId `
        -ProductIdFile $ProductIdFile `
        -DurationMinutes $DurationMinutes
    if (-not $?) { exit 1 }
    $ActivityId = [long](Get-Content -Raw .memory/runlogs/perf.activity_id.txt).Trim()
    $ItemId = [long](Get-Content -Raw .memory/runlogs/perf.item_id.txt).Trim()
}

if ($ActivityId -le 0 -or $ItemId -le 0) {
    throw "ActivityId 和 ItemId 必须大于 0；可加 -Prepare 自动创建压测活动。"
}

$idemGroup = "idem-$([DateTimeOffset]::UtcNow.ToUnixTimeMilliseconds())"
$idemToken = $Token
if ([string]::IsNullOrWhiteSpace($idemToken) -and -not [string]::IsNullOrWhiteSpace($TokenFile)) {
    if (-not (Test-Path $TokenFile)) {
        throw "TokenFile 不存在: $TokenFile"
    }
    $idemToken = (Get-Content $TokenFile | Where-Object { -not [string]::IsNullOrWhiteSpace($_) -and -not $_.Trim().StartsWith("#") } | Select-Object -First 1)
}
if ([string]::IsNullOrWhiteSpace($idemToken)) {
    throw "幂等场景需要有效用户 token（可通过 -Token 或 -TokenFile 提供）"
}

Write-Host "=== 场景1: 幂等重放冲突测试（同幂等键并发） ==="
& (Join-Path $repoRoot "scripts/dev/seckill/seckill-perf.ps1") `
    -Scenario "idempotency" `
    -BaseUrl $BaseUrl `
    -ActivityId $ActivityId `
    -ItemId $ItemId `
    -IdempotencyGroup $idemGroup `
    -Concurrency ([Math]::Min($Concurrency, 50)) `
    -Requests ([Math]::Min($Requests, 200)) `
    -Token $idemToken
if (-not $?) { exit 1 }

Write-Host "=== 场景2: 抢购压力测试（库存/限购冲突） ==="
& (Join-Path $repoRoot "scripts/dev/seckill/seckill-perf.ps1") `
    -Scenario "purchase-stress" `
    -BaseUrl $BaseUrl `
    -ActivityId $ActivityId `
    -ItemId $ItemId `
    -Concurrency $Concurrency `
    -Requests $Requests `
    -Token $Token `
    -TokenFile $TokenFile `
    -ExpectMaxSuccess $ExpectedMaxSuccess
if (-not $?) { exit 1 }

Write-Host "=== 场景3: 匿名埋点洪峰测试 ==="
& (Join-Path $repoRoot "scripts/dev/seckill/seckill-perf.ps1") `
    -Scenario "track-stress" `
    -BaseUrl $BaseUrl `
    -ActivityId $ActivityId `
    -ItemId $ItemId `
    -Concurrency $Concurrency `
    -Requests $Requests `
    -MaxConnsPerHost $TrackMaxConnsPerHost
if (-not $?) { exit 1 }


