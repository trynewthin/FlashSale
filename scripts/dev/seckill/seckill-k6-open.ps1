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
    [long]$ActivityId = 0,
    [long]$ItemId = 0,
    [string]$TokenFile = ".memory/runlogs/user.tokens.txt",
    [int]$MinTokenCount = 120,
    [int]$PurchaseRate = 120,
    [int]$TrackRate = 0,
    [int]$DurationSeconds = 30,
    [int]$Quantity = 1,
    [int]$PurchaseMaxP95Ms = 5000,
    [double]$PurchaseMinEffectiveRate = 0.95,
    [double]$PurchaseMaxNetworkRate = 0.03,
    [int]$TrackMaxP95Ms = 500,
    [double]$TrackMinEffectiveRate = 0.99,
    [double]$TrackMaxNetworkRate = 0.01,
    [string]$EventType = "pv",
    [string]$ClientIdPrefix = "k6-client",
    [string]$OutputRoot = ".memory/runlogs",
    [string]$DockerImage = "grafana/k6:0.57.0",
    [bool]$UseDocker = $true
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

function Invoke-NativeCommand {
    param(
        [string]$Exe,
        [string[]]$CmdArgs,
        [string]$LogPath
    )
    $safeArgs = @()
    foreach ($arg in $CmdArgs) {
        if ($null -eq $arg) {
            continue
        }
        $text = "$arg"
        if ([string]::IsNullOrWhiteSpace($text)) {
            continue
        }
        $safeArgs += $text
    }
    $lines = @()
    $lines += "[cmd] $Exe $($safeArgs -join ' ')"
    $oldPreference = $ErrorActionPreference
    $ErrorActionPreference = "Continue"
    & $Exe @safeArgs 2>&1 | ForEach-Object {
        $lines += "$_"
    }
    $exitCode = $LASTEXITCODE
    $ErrorActionPreference = $oldPreference
    Set-Content -Path $LogPath -Value $lines
    return [pscustomobject]@{
        ExitCode = [int]$exitCode
        Lines = $lines
    }
}

function Write-Info {
    param([string]$Message)
    Write-Host "[k6-open] $Message"
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

function Resolve-ActivityItem {
    param(
        [long]$InputActivityId,
        [long]$InputItemId
    )
    $aid = $InputActivityId
    $iid = $InputItemId
    if ($aid -le 0) {
        $aid = [long](Get-Content -Raw .memory/runlogs/perf.activity_id.txt).Trim()
    }
    if ($iid -le 0) {
        $iid = [long](Get-Content -Raw .memory/runlogs/perf.item_id.txt).Trim()
    }
    if ($aid -le 0 -or $iid -le 0) {
        throw "invalid activity/item id"
    }
    return [pscustomobject]@{
        activity_id = $aid
        item_id = $iid
    }
}

function To-DockerBaseUrl {
    param([string]$Url)
    if ([string]::IsNullOrWhiteSpace($Url)) {
        return $Url
    }
    $u = $Url
    $u = $u -replace "127\.0\.0\.1", "host.docker.internal"
    $u = $u -replace "localhost", "host.docker.internal"
    return $u
}

Write-Info "precheck"
Assert-Healthy -Url $BaseUrl
Assert-Healthy -Url $AdminBaseUrl
$tokenCount = Ensure-TokenPool -Path $TokenFile -MinCount $MinTokenCount
Write-Info "token pool ready: $tokenCount"

if ($Prepare) {
    $adminAccessToken = Resolve-AdminToken -Base $AdminBaseUrl -Token $AdminToken -TokenFile $AdminTokenFile -Username $AdminUsername -Password $AdminPassword
    Write-Info "prepare fresh activity/item"
    powershell -ExecutionPolicy Bypass -File (Join-Path $repoRoot "scripts/dev/seckill/seckill-prepare.ps1") `
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

$ids = Resolve-ActivityItem -InputActivityId $ActivityId -InputItemId $ItemId
$timestamp = Get-Date -Format "yyyyMMdd-HHmmss"
$outputDir = Join-Path $OutputRoot "k6-open-$timestamp"
New-Item -ItemType Directory -Path $outputDir -Force | Out-Null
$summaryPath = Join-Path $outputDir "summary.json"
$rawLog = Join-Path $outputDir "run.log"

$repoPath = (Resolve-Path ".").Path
$k6ScriptPath = "scripts/perf/k6/seckill_open_model.js"
$runExitCode = 0

$envPairs = @{
    BASE_URL = $(if($UseDocker){To-DockerBaseUrl -Url $BaseUrl}else{$BaseUrl})
    ACTIVITY_ID = "$($ids.activity_id)"
    ITEM_ID = "$($ids.item_id)"
    TOKEN_FILE = $(if($UseDocker){"/work/$($TokenFile -replace '\\','/')"}else{(Resolve-Path $TokenFile).Path})
    TEST_DURATION = "$($DurationSeconds)s"
    PURCHASE_RATE = "$PurchaseRate"
    TRACK_RATE = "$TrackRate"
    QUANTITY = "$Quantity"
    EVENT_TYPE = "$EventType"
    CLIENT_ID_PREFIX = "$ClientIdPrefix"
    PURCHASE_MAX_P95_MS = "$PurchaseMaxP95Ms"
    PURCHASE_MIN_EFFECTIVE_RATE = "$PurchaseMinEffectiveRate"
    PURCHASE_MAX_NETWORK_RATE = "$PurchaseMaxNetworkRate"
    TRACK_MAX_P95_MS = "$TrackMaxP95Ms"
    TRACK_MIN_EFFECTIVE_RATE = "$TrackMinEffectiveRate"
    TRACK_MAX_NETWORK_RATE = "$TrackMaxNetworkRate"
    SUMMARY_PATH = $(if($UseDocker){"/work/$($summaryPath -replace '\\','/')"}else{$summaryPath})
}

if (-not $UseDocker) {
    $k6 = Get-Command k6 -ErrorAction SilentlyContinue
    if ($null -eq $k6) {
        throw "k6 not found, set -UseDocker true or install k6"
    }
    $k6Args = @("run")
    foreach ($key in $envPairs.Keys) {
        $k6Args += @("-e", "$key=$($envPairs[$key])")
    }
    $k6Args += $k6ScriptPath
    Write-Info "running k6 locally"
    $run = Invoke-NativeCommand -Exe "k6" -CmdArgs $k6Args -LogPath $rawLog
    $runExitCode = [int]$run.ExitCode
} else {
    $docker = Get-Command docker -ErrorAction SilentlyContinue
    if ($null -eq $docker) {
        throw "docker not found"
    }
    $dockerArgs = @("run", "--rm", "-v", "${repoPath}:/work", "-w", "/work")
    foreach ($key in $envPairs.Keys) {
        $dockerArgs += @("-e", "$key=$($envPairs[$key])")
    }
    $dockerArgs += @($DockerImage, "run", "/work/$($k6ScriptPath -replace '\\','/')")
    Write-Info "running k6 via docker image $DockerImage"
    $run = Invoke-NativeCommand -Exe "docker" -CmdArgs $dockerArgs -LogPath $rawLog
    $runExitCode = [int]$run.ExitCode
}

if (-not (Test-Path $summaryPath)) {
    if ($runExitCode -ne 0) {
        throw "k6 failed($runExitCode), summary missing, log: $rawLog"
    }
    throw "summary file not found: $summaryPath"
}

$summary = Get-Content -Raw $summaryPath | ConvertFrom-Json
$reportPath = Join-Path $outputDir "report.md"
$sb = New-Object System.Text.StringBuilder
[void]$sb.AppendLine("# k6 Open-Model Report")
[void]$sb.AppendLine("")
[void]$sb.AppendLine("- GeneratedAt: $(Get-Date -Format 'yyyy-MM-dd HH:mm:ss')")
[void]$sb.AppendLine("- BaseUrl: $($summary.config.base_url)")
[void]$sb.AppendLine("- Activity/Item: $($summary.config.activity_id) / $($summary.config.item_id)")
[void]$sb.AppendLine("- Duration: $($summary.config.duration)")
[void]$sb.AppendLine("- PurchaseRate: $($summary.config.purchase_rate) req/s")
[void]$sb.AppendLine("- TrackRate: $($summary.config.track_rate) req/s")
[void]$sb.AppendLine("- k6_exit_code: $runExitCode")
[void]$sb.AppendLine("")
[void]$sb.AppendLine("## Key Metrics")
[void]$sb.AppendLine("")
[void]$sb.AppendLine("| Metric | Value |")
[void]$sb.AppendLine("|---|---:|")
[void]$sb.AppendLine("| http_req_duration_p95_ms | $([math]::Round([double]$summary.result.http_req_duration_p95,2)) |")
[void]$sb.AppendLine("| http_req_duration_p99_ms | $([math]::Round([double]$summary.result.http_req_duration_p99,2)) |")
[void]$sb.AppendLine("| purchase_effective_rate | $([math]::Round([double]$summary.result.purchase_effective_rate,4)) |")
[void]$sb.AppendLine("| purchase_ok_rate | $([math]::Round([double]$summary.result.purchase_ok_rate,4)) |")
[void]$sb.AppendLine("| purchase_network_error_rate | $([math]::Round([double]$summary.result.purchase_network_error_rate,4)) |")
[void]$sb.AppendLine("| purchase_http_5xx_rate | $([math]::Round([double]$summary.result.purchase_http_5xx_rate,4)) |")
[void]$sb.AppendLine("| track_effective_rate | $([math]::Round([double]$summary.result.track_effective_rate,4)) |")
[void]$sb.AppendLine("| track_accepted_rate | $([math]::Round([double]$summary.result.track_accepted_rate,4)) |")
[void]$sb.AppendLine("| track_network_error_rate | $([math]::Round([double]$summary.result.track_network_error_rate,4)) |")
[void]$sb.AppendLine("| track_http_5xx_rate | $([math]::Round([double]$summary.result.track_http_5xx_rate,4)) |")
[void]$sb.AppendLine("")
[void]$sb.AppendLine("## Outputs")
[void]$sb.AppendLine("- summary: $summaryPath")
[void]$sb.AppendLine("- raw log: $rawLog")
[void]$sb.AppendLine("- report: $reportPath")
Set-Content -Path $reportPath -Value $sb.ToString()

Write-Info "report generated: $reportPath"
Write-Info "summary generated: $summaryPath"


