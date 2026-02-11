param(
    [ValidateSet("purchase", "track")]
    [string]$Mode = "purchase",
    [string]$Rates = "80,120,160,200",
    [int]$DurationSeconds = 15,
    [string]$BaseUrl = "http://127.0.0.1:8082",
    [string]$AdminBaseUrl = "http://127.0.0.1:8083",
    [bool]$Prepare = $true,
    [bool]$AutoLoadDevEnv = $true,
    [string]$EnvFile = "configs/local/dev.env",
    [string]$AdminToken = "",
    [string]$AdminTokenFile = ".memory/runlogs/admin.token.txt",
    [string]$AdminUsername = $env:FLASHSALE_ADMIN_USERNAME,
    [string]$AdminPassword = $env:FLASHSALE_ADMIN_PASSWORD,
    [long]$ProductId = 0,
    [string]$ProductIdFile = ".memory/runlogs/perf.product_id.txt",
    [int]$DurationMinutes = 45,
    [long]$ReservedStockTotal = 20000,
    [string]$TokenFile = ".memory/runlogs/user.tokens.txt",
    [int]$MinTokenCount = 120,
    [double]$MinEffectiveRate = 0.95,
    [double]$MaxNetworkRate = 0.03,
    [int]$MaxP95Ms = 5000,
    [string]$OutputRoot = ".memory/runlogs"
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

function Write-Info {
    param([string]$Message)
    Write-Host "[k6-ladder] $Message"
}

function Parse-Rates {
    param([string]$Raw)
    $result = New-Object System.Collections.Generic.List[int]
    foreach ($part in $Raw.Split(",")) {
        $trimmed = $part.Trim()
        if ([string]::IsNullOrWhiteSpace($trimmed)) {
            continue
        }
        $parsed = 0
        if ([int]::TryParse($trimmed, [ref]$parsed) -and $parsed -gt 0) {
            $result.Add($parsed) | Out-Null
        }
    }
    if ($result.Count -eq 0) {
        throw "invalid Rates: $Raw"
    }
    return $result
}

function Get-LatestK6OpenDir {
    param([datetime]$AfterTime)
    return Get-ChildItem .memory/runlogs -Directory |
        Where-Object { $_.Name -like "k6-open-*" -and $_.LastWriteTime -ge $AfterTime } |
        Sort-Object LastWriteTime -Descending |
        Select-Object -First 1
}

function Read-K6Summary {
    param([string]$Path)
    if (-not (Test-Path $Path)) {
        throw "summary not found: $Path"
    }
    return Get-Content -Raw $Path | ConvertFrom-Json
}

function Evaluate-Stage {
    param(
        [object]$Summary,
        [string]$TargetMode,
        [double]$StageMinEffectiveRate,
        [double]$StageMaxNetworkRate,
        [int]$StageMaxP95Ms
    )
    $r = $Summary.result
    if ($TargetMode -eq "purchase") {
        $effective = [double]$r.purchase_effective_rate
        $network = [double]$r.purchase_network_error_rate
    } else {
        $effective = [double]$r.track_effective_rate
        $network = [double]$r.track_network_error_rate
    }
    $p95 = [double]$r.http_req_duration_p95
    $pass = ($effective -ge $StageMinEffectiveRate -and $network -le $StageMaxNetworkRate -and $p95 -le $StageMaxP95Ms)
    $reasons = New-Object System.Collections.Generic.List[string]
    if ($effective -lt $StageMinEffectiveRate) { $reasons.Add("effective_rate=$([math]::Round($effective,4)) < $StageMinEffectiveRate") }
    if ($network -gt $StageMaxNetworkRate) { $reasons.Add("network_rate=$([math]::Round($network,4)) > $StageMaxNetworkRate") }
    if ($p95 -gt $StageMaxP95Ms) { $reasons.Add("p95=$([math]::Round($p95,2))ms > ${StageMaxP95Ms}ms") }
    return [pscustomobject]@{
        pass = $pass
        effective_rate = $effective
        network_rate = $network
        p95_ms = $p95
        reasons = ($reasons -join "; ")
    }
}

$timestamp = Get-Date -Format "yyyyMMdd-HHmmss"
$outputDir = Join-Path $OutputRoot "k6-ladder-$timestamp"
New-Item -ItemType Directory -Path $outputDir -Force | Out-Null

$rateList = Parse-Rates -Raw $Rates
$rows = New-Object System.Collections.Generic.List[object]

$activityId = 0L
$itemId = 0L
$index = 0

foreach ($rate in $rateList) {
    $before = Get-Date
    $purchaseRate = if ($Mode -eq "purchase") { $rate } else { 0 }
    $trackRate = if ($Mode -eq "track") { $rate } else { 0 }
    $prepareThisRound = ($index -eq 0 -and $Prepare)
    Write-Info "run mode=$Mode rate=$rate prepare=$prepareThisRound"

    $invokeParams = @{
        BaseUrl = $BaseUrl
        AdminBaseUrl = $AdminBaseUrl
        Prepare = $prepareThisRound
        ProductId = $ProductId
        ProductIdFile = $ProductIdFile
        DurationMinutes = $DurationMinutes
        ReservedStockTotal = $ReservedStockTotal
        TokenFile = $TokenFile
        MinTokenCount = $MinTokenCount
        PurchaseRate = $purchaseRate
        TrackRate = $trackRate
        DurationSeconds = $DurationSeconds
        PurchaseMinEffectiveRate = $MinEffectiveRate
        PurchaseMaxNetworkRate = $MaxNetworkRate
        PurchaseMaxP95Ms = $MaxP95Ms
    }
    if (-not [string]::IsNullOrWhiteSpace($AdminToken)) {
        $invokeParams.AdminToken = $AdminToken
    }
    if (-not [string]::IsNullOrWhiteSpace($AdminTokenFile)) {
        $invokeParams.AdminTokenFile = $AdminTokenFile
    }
    if (-not [string]::IsNullOrWhiteSpace($AdminUsername)) {
        $invokeParams.AdminUsername = $AdminUsername
    }
    if (-not [string]::IsNullOrWhiteSpace($AdminPassword)) {
        $invokeParams.AdminPassword = $AdminPassword
    }
    if (-not $prepareThisRound) {
        $invokeParams.ActivityId = $activityId
        $invokeParams.ItemId = $itemId
    }

    & ./scripts/dev/seckill-k6-open.ps1 @invokeParams

    $latest = Get-LatestK6OpenDir -AfterTime $before
    if ($null -eq $latest) {
        throw "cannot find k6-open output for rate=$rate"
    }
    $summaryPath = Join-Path $latest.FullName "summary.json"
    $reportPath = Join-Path $latest.FullName "report.md"
    $summary = Read-K6Summary -Path $summaryPath
    if ($index -eq 0) {
        $activityId = [long]$summary.config.activity_id
        $itemId = [long]$summary.config.item_id
    }

    $eval = Evaluate-Stage -Summary $summary -TargetMode $Mode -StageMinEffectiveRate $MinEffectiveRate -StageMaxNetworkRate $MaxNetworkRate -StageMaxP95Ms $MaxP95Ms
    $rows.Add([pscustomobject]@{
        rate = $rate
        pass = $eval.pass
        effective_rate = $eval.effective_rate
        network_rate = $eval.network_rate
        p95_ms = $eval.p95_ms
        reasons = $eval.reasons
        summary_path = $summaryPath
        report_path = $reportPath
    }) | Out-Null
    $index++
}

$best = $rows | Where-Object { $_.pass } | Sort-Object rate -Descending | Select-Object -First 1
$recommendedRate = if ($null -eq $best) { 0 } else { [int]$best.rate }

$reportOut = Join-Path $outputDir "ladder-report.md"
$sb = New-Object System.Text.StringBuilder
[void]$sb.AppendLine("# k6 Ladder Report")
[void]$sb.AppendLine("")
[void]$sb.AppendLine("- GeneratedAt: $(Get-Date -Format 'yyyy-MM-dd HH:mm:ss')")
[void]$sb.AppendLine("- Mode: $Mode")
[void]$sb.AppendLine("- BaseUrl: $BaseUrl")
[void]$sb.AppendLine("- Activity/Item: $activityId / $itemId")
[void]$sb.AppendLine("- Duration: ${DurationSeconds}s")
[void]$sb.AppendLine("- Thresholds: effective_rate >= $MinEffectiveRate, network_rate <= $MaxNetworkRate, p95 <= ${MaxP95Ms}ms")
[void]$sb.AppendLine("- RecommendedRate: $recommendedRate req/s")
[void]$sb.AppendLine("")
[void]$sb.AppendLine("| Rate(req/s) | Pass | EffectiveRate | NetworkRate | P95(ms) | Notes |")
[void]$sb.AppendLine("|---:|---|---:|---:|---:|---|")
foreach ($row in ($rows | Sort-Object rate)) {
    $passText = if ($row.pass) { "PASS" } else { "FAIL" }
    $notes = if ([string]::IsNullOrWhiteSpace($row.reasons)) { "-" } else { $row.reasons }
    [void]$sb.AppendLine("| $($row.rate) | $passText | $([math]::Round([double]$row.effective_rate,4)) | $([math]::Round([double]$row.network_rate,4)) | $([math]::Round([double]$row.p95_ms,2)) | $notes |")
}
[void]$sb.AppendLine("")
[void]$sb.AppendLine("## Stage Outputs")
foreach ($row in $rows) {
    [void]$sb.AppendLine("- rate=$($row.rate): report=$($row.report_path), summary=$($row.summary_path)")
}
Set-Content -Path $reportOut -Value $sb.ToString()

$jsonOut = Join-Path $outputDir "ladder-summary.json"
@{
    generated_at = (Get-Date).ToString("s")
    mode = $Mode
    base_url = $BaseUrl
    activity_id = $activityId
    item_id = $itemId
    duration_seconds = $DurationSeconds
    thresholds = @{
        min_effective_rate = $MinEffectiveRate
        max_network_rate = $MaxNetworkRate
        max_p95_ms = $MaxP95Ms
    }
    recommended_rate = $recommendedRate
    rows = $rows
} | ConvertTo-Json -Depth 10 | Set-Content -Path $jsonOut

Write-Info "report generated: $reportOut"
Write-Info "summary generated: $jsonOut"
