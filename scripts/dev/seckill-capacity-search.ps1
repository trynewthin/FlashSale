param(
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
    [string]$ConcurrencyList = "80,120,160,200,240,280,320",
    [int]$RequestsPerStage = 1600,
    [int]$TimeoutMs = 7000,
    [int]$MaxConnsPerHost = 220,
    [double]$MinSuccessRate = 0.95,
    [double]$MaxNetworkErrorRate = 0.03,
    [int]$MaxP95Ms = 5000,
    [int]$MaxP99Ms = 7000,
    [ValidateSet("go-run", "binary")]
    [string]$LoadRunnerMode = "binary",
    [string]$LoadBinaryPath = ".memory/runlogs/bin/seckillload.exe",
    [bool]$BuildLoadBinary = $true,
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
    Write-Host "[capacity] $Message"
}

function Assert-Healthy {
    param([string]$Url)
    $resp = Invoke-WebRequest -UseBasicParsing -TimeoutSec 2 "$Url/healthz"
    if ($resp.StatusCode -ne 200) {
        throw "health check failed: $Url => $($resp.StatusCode)"
    }
}

function Ensure-TokenPool {
    param([string]$Path, [int]$MinCount)
    $count = 0
    if (Test-Path $Path) {
        $count = (Get-Content $Path | Where-Object { -not [string]::IsNullOrWhiteSpace($_) }).Count
    }
    if ($count -ge $MinCount) {
        return $count
    }
    Write-Info "token pool insufficient ($count/$MinCount), regenerating"
    powershell -ExecutionPolicy Bypass -File .memory/runlogs/gen_tokens.ps1 | Out-Null
    $count = (Get-Content $Path | Where-Object { -not [string]::IsNullOrWhiteSpace($_) }).Count
    if ($count -lt $MinCount) {
        throw "token pool insufficient after regenerate: $count/$MinCount"
    }
    return $count
}

function Resolve-AdminToken {
    param(
        [string]$Base,
        [string]$Token,
        [string]$TokenFile,
        [string]$Username,
        [string]$Password
    )
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

    $canLogin = (-not [string]::IsNullOrWhiteSpace($Username) -and -not [string]::IsNullOrWhiteSpace($Password))
    $login = {
        $body = @{ username = $Username.Trim(); password = $Password.Trim() } | ConvertTo-Json -Compress
        $resp = Invoke-RestMethod -Method Post -Uri "$($Base.TrimEnd('/'))/api/v1/admin/auth/login" -ContentType "application/json" -Body $body
        if ($resp.code -ne "OK" -or [string]::IsNullOrWhiteSpace($resp.data.access_token)) {
            throw "admin login failed"
        }
        return $resp.data.access_token.Trim()
    }

    if (-not [string]::IsNullOrWhiteSpace($Token)) {
        $trimmed = $Token.Trim()
        if ((& $validateToken $trimmed)) {
            return $trimmed
        }
        if ($canLogin) {
            Write-Info "admin token invalid, fallback to username/password login"
            return (& $login)
        }
        throw "admin token invalid, and no username/password provided"
    }

    if (Test-Path $TokenFile) {
        $raw = (Get-Content -Raw $TokenFile).Trim()
        if (-not [string]::IsNullOrWhiteSpace($raw)) {
            if ((& $validateToken $raw)) {
                return $raw
            }
            if ($canLogin) {
                Write-Info "admin token file expired, fallback to username/password login"
                return (& $login)
            }
            throw "admin token from file is invalid: $TokenFile"
        }
    }

    if ($canLogin) {
        return (& $login)
    }
    throw "admin token missing and username/password not provided"
}

function Ensure-SeckillLoadBinary {
    param([string]$Path)
    $dir = Split-Path -Parent $Path
    if (-not [string]::IsNullOrWhiteSpace($dir) -and -not (Test-Path $dir)) {
        New-Item -ItemType Directory -Path $dir -Force | Out-Null
    }
    go build -o $Path ./cmd/perf/seckillload
    if ($LASTEXITCODE -ne 0) {
        throw "build seckillload failed"
    }
}

function Invoke-SeckillLoadJson {
    param([string[]]$ToolArgs, [string]$RawLogFile)
    $lines = @()
    if ($LoadRunnerMode -eq "binary") {
        & $LoadBinaryPath @ToolArgs 2>&1 | ForEach-Object { $lines += "$_" }
    } else {
        $goArgs = @("run", "./cmd/perf/seckillload") + $ToolArgs
        & go @goArgs 2>&1 | ForEach-Object { $lines += "$_" }
    }
    $exitCode = $LASTEXITCODE
    Set-Content -Path $RawLogFile -Value $lines
    if ($exitCode -ne 0) {
        throw "seckillload failed($exitCode), log: $RawLogFile"
    }
    $jsonLine = $lines | Where-Object { $_.Trim().StartsWith("{") -and $_.Trim().EndsWith("}") } | Select-Object -Last 1
    if ([string]::IsNullOrWhiteSpace($jsonLine)) {
        throw "json output not found, log: $RawLogFile"
    }
    return ($jsonLine | ConvertFrom-Json)
}

function Parse-ConcurrencyList {
    param([string]$Raw)
    $list = New-Object System.Collections.Generic.List[int]
    foreach ($part in $Raw.Split(",")) {
        $trimmed = $part.Trim()
        if ([string]::IsNullOrWhiteSpace($trimmed)) {
            continue
        }
        $parsed = 0
        if ([int]::TryParse($trimmed, [ref]$parsed) -and $parsed -gt 0) {
            $list.Add($parsed) | Out-Null
        }
    }
    if ($list.Count -eq 0) {
        throw "invalid ConcurrencyList: $Raw"
    }
    return $list
}

function Test-Thresholds {
    param([object]$Summary)
    $reasons = New-Object System.Collections.Generic.List[string]
    $pass = $true
    if ([double]$Summary.network_error_rate -gt $MaxNetworkErrorRate) {
        $pass = $false
        $reasons.Add("network_error_rate=$([math]::Round($Summary.network_error_rate,4)) > $MaxNetworkErrorRate")
    }
    if ([double]$Summary.success_rate -lt $MinSuccessRate) {
        $pass = $false
        $reasons.Add("success_rate=$([math]::Round($Summary.success_rate,4)) < $MinSuccessRate")
    }
    if ([double]$Summary.latency_p95_ms -gt $MaxP95Ms) {
        $pass = $false
        $reasons.Add("p95=$([math]::Round($Summary.latency_p95_ms,2))ms > ${MaxP95Ms}ms")
    }
    if ([double]$Summary.latency_p99_ms -gt $MaxP99Ms) {
        $pass = $false
        $reasons.Add("p99=$([math]::Round($Summary.latency_p99_ms,2))ms > ${MaxP99Ms}ms")
    }
    return [pscustomobject]@{
        pass    = $pass
        reasons = $reasons
    }
}

$timestamp = Get-Date -Format "yyyyMMdd-HHmmss"
$outputDir = Join-Path $OutputRoot "capacity-$timestamp"
New-Item -ItemType Directory -Path $outputDir -Force | Out-Null

Write-Info "precheck"
Assert-Healthy -Url $BaseUrl
Assert-Healthy -Url $AdminBaseUrl
$tokenCount = Ensure-TokenPool -Path $TokenFile -MinCount $MinTokenCount
$adminAccessToken = Resolve-AdminToken -Base $AdminBaseUrl -Token $AdminToken -TokenFile $AdminTokenFile -Username $AdminUsername -Password $AdminPassword
Write-Info "token pool ready: $tokenCount"

if ($LoadRunnerMode -eq "binary") {
    if ($BuildLoadBinary -or -not (Test-Path $LoadBinaryPath)) {
        Write-Info "building seckillload binary"
        Ensure-SeckillLoadBinary -Path $LoadBinaryPath
    }
}

if ($Prepare) {
    Write-Info "prepare fresh activity/item"
    powershell -ExecutionPolicy Bypass -File scripts/dev/seckill-prepare.ps1 `
        -BaseUrl $AdminBaseUrl `
        -AdminToken $adminAccessToken `
        -ProductId $ProductId `
        -ProductIdFile $ProductIdFile `
        -DurationMinutes $DurationMinutes `
        -ReservedStockTotal $ReservedStockTotal
    if ($LASTEXITCODE -ne 0) {
        throw "prepare failed"
    }
}

[long]$activityID = (Get-Content -Raw .memory/runlogs/perf.activity_id.txt).Trim()
[long]$itemID = (Get-Content -Raw .memory/runlogs/perf.item_id.txt).Trim()
if ($activityID -le 0 -or $itemID -le 0) {
    throw "invalid activity/item id"
}

$stages = New-Object System.Collections.Generic.List[object]
$concurrencyLevels = Parse-ConcurrencyList -Raw $ConcurrencyList

foreach ($c in $concurrencyLevels) {
    Write-Info "running c=$c r=$RequestsPerStage"
    $args = @(
        "-scenario", "purchase-stress",
        "-base-url", $BaseUrl,
        "-activity-id", "$activityID",
        "-item-id", "$itemID",
        "-concurrency", "$c",
        "-requests", "$RequestsPerStage",
        "-timeout", "$($TimeoutMs)ms",
        "-max-idle-conns", "1024",
        "-max-idle-conns-per-host", "512",
        "-max-conns-per-host", "$MaxConnsPerHost",
        "-output", "json",
        "-token-file", $TokenFile,
        "-max-network-errors", "999999"
    )
    $rawLog = Join-Path $outputDir "purchase-c$c.raw.log"
    $json = Invoke-SeckillLoadJson -ToolArgs $args -RawLogFile $rawLog
    $check = Test-Thresholds -Summary $json.summary
    $stage = [pscustomobject]@{
        concurrency        = $c
        requests           = [int]$json.config.requests
        pass               = $check.pass
        reasons            = ($check.reasons -join "; ")
        success_rate       = [double]$json.summary.success_rate
        network_error_rate = [double]$json.summary.network_error_rate
        p95_ms             = [double]$json.summary.latency_p95_ms
        p99_ms             = [double]$json.summary.latency_p99_ms
        rps                = [double]$json.summary.rps
        raw_log            = $rawLog
    }
    $stages.Add($stage) | Out-Null
    ($json | ConvertTo-Json -Depth 10) | Set-Content (Join-Path $outputDir "purchase-c$c.json")
}

$best = $stages | Where-Object { $_.pass } | Sort-Object concurrency -Descending | Select-Object -First 1
$bestConcurrency = if ($null -eq $best) { 0 } else { [int]$best.concurrency }

$reportPath = Join-Path $outputDir "capacity-report.md"
$sb = New-Object System.Text.StringBuilder
[void]$sb.AppendLine("# Seckill Capacity Search Report")
[void]$sb.AppendLine("")
[void]$sb.AppendLine("- GeneratedAt: $(Get-Date -Format 'yyyy-MM-dd HH:mm:ss')")
[void]$sb.AppendLine("- BaseUrl: $BaseUrl")
[void]$sb.AppendLine("- Activity/Item: $activityID / $itemID")
[void]$sb.AppendLine("- Thresholds: success_rate >= $MinSuccessRate, network_error_rate <= $MaxNetworkErrorRate, p95 <= ${MaxP95Ms}ms, p99 <= ${MaxP99Ms}ms")
[void]$sb.AppendLine("- RecommendedStableConcurrency: $bestConcurrency")
[void]$sb.AppendLine("")
[void]$sb.AppendLine("| Concurrency | Pass | SuccessRate | NetworkErrRate | P95(ms) | P99(ms) | RPS | Notes |")
[void]$sb.AppendLine("|---:|---|---:|---:|---:|---:|---:|---|")
foreach ($s in ($stages | Sort-Object concurrency)) {
    $passText = if ($s.pass) { "PASS" } else { "FAIL" }
    $notes = if ([string]::IsNullOrWhiteSpace($s.reasons)) { "-" } else { $s.reasons }
    [void]$sb.AppendLine("| $($s.concurrency) | $passText | $([math]::Round($s.success_rate*100,2))% | $([math]::Round($s.network_error_rate*100,2))% | $([math]::Round($s.p95_ms,2)) | $([math]::Round($s.p99_ms,2)) | $([math]::Round($s.rps,2)) | $notes |")
}
[void]$sb.AppendLine("")
[void]$sb.AppendLine("## Raw Outputs")
foreach ($s in $stages) {
    [void]$sb.AppendLine("- c$($s.concurrency): $($s.raw_log)")
}
[void]$sb.AppendLine("- report: $reportPath")
Set-Content -Path $reportPath -Value $sb.ToString()

$summaryPath = Join-Path $outputDir "capacity-summary.json"
@{
    generated_at = (Get-Date).ToString("s")
    activity_id = $activityID
    item_id = $itemID
    best_concurrency = $bestConcurrency
    thresholds = @{
        min_success_rate = $MinSuccessRate
        max_network_error_rate = $MaxNetworkErrorRate
        max_p95_ms = $MaxP95Ms
        max_p99_ms = $MaxP99Ms
    }
    stages = $stages
} | ConvertTo-Json -Depth 10 | Set-Content -Path $summaryPath

Write-Info "report generated: $reportPath"
Write-Info "best stable concurrency: $bestConcurrency"
if ($bestConcurrency -le 0) {
    exit 2
}
exit 0
