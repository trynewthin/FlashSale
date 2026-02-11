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
    [long]$UserLimitQty = 0,
    [string]$TokenFile = ".memory/runlogs/user.tokens.txt",
    [int]$MinTokenCount = 120,
    [string]$OutputRoot = ".memory/runlogs",
    [int]$L2PurchaseTimeoutMs = 6000,
    [int]$L2PurchaseMaxConnsPerHost = 200,
    [int]$L2PurchaseConcurrency = 200,
    [int]$L2PurchaseRequests = 4000,
    [double]$L2MinSuccessRate = 0.92,
    [double]$L2MaxNetworkErrorRate = 0.05,
    [int]$L2MaxP95Ms = 5200,
    [int]$L2MaxP99Ms = 6500,
    [int]$L2C100Concurrency = 100,
    [int]$L2C100Requests = 2000,
    [int]$L2C100TimeoutMs = 6000,
    [int]$L2C100MaxConnsPerHost = 120,
    [double]$L2C100MinSuccessRate = 0.97,
    [double]$L2C100MaxNetworkErrorRate = 0.02,
    [int]$L2C100MaxP95Ms = 4500,
    [int]$L2C100MaxP99Ms = 6000,
    [int]$L3PurchaseTimeoutMs = 7000,
    [int]$L3PurchaseMaxConnsPerHost = 220,
    [int]$L3PurchaseConcurrency = 200,
    [int]$L3PurchaseRequests = 4000,
    [double]$L3MinSuccessRate = 0.95,
    [double]$L3MaxNetworkErrorRate = 0.05,
    [int]$L3MaxP95Ms = 6500,
    [int]$L3MaxP99Ms = 7800,
    [int]$TrackTimeoutMs = 3000,
    [int]$TrackMaxConnsPerHost = 200,
    [int]$StageCooldownSeconds = 6,
    [bool]$EnableStageStabilize = $true,
    [int]$StageStabilizeProbeConcurrency = 10,
    [int]$StageStabilizeProbeRequests = 60,
    [double]$StageStabilizeMinSuccessRate = 0.95,
    [int]$StageStabilizeMaxWaitSeconds = 45,
    [bool]$CleanupStalePerfActivities = $true,
    [bool]$CleanupCurrentActivity = $true,
    [bool]$EnableL3 = $false,
    [ValidateSet("stable", "strict")]
    [string]$GateProfile = "stable",
    [string]$StopOnFirstFailure = "true",
    [string]$AutoDiagnoseOnL2Failure = "true",
    [ValidateSet("go-run", "binary")]
    [string]$LoadRunnerMode = "binary",
    [string]$LoadBinaryPath = ".memory/runlogs/bin/seckillload.exe",
    [bool]$BuildLoadBinary = $true,
    [bool]$RestartRuntimeBeforeRun = $false,
    [bool]$RuntimeForceIPv4Loopback = $true,
    [int]$RuntimeMySQLMaxOpenConns = 32,
    [int]$RuntimeMySQLMaxIdleConns = 8,
    [int]$RuntimeReservePurchaseTimeoutMs = 5000
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

function Parse-Bool {
    param(
        [string]$Raw,
        [bool]$DefaultValue
    )
    $v = "$Raw".Trim().ToLowerInvariant()
    if ([string]::IsNullOrWhiteSpace($v)) {
        return $DefaultValue
    }
    switch ($v) {
        "1" { return $true }
        "true" { return $true }
        "yes" { return $true }
        "on" { return $true }
        "0" { return $false }
        "false" { return $false }
        "no" { return $false }
        "off" { return $false }
        default { return $DefaultValue }
    }
}

$stopOnFirstFailureEnabled = Parse-Bool -Raw $StopOnFirstFailure -DefaultValue $true
$autoDiagnoseOnL2FailureEnabled = Parse-Bool -Raw $AutoDiagnoseOnL2Failure -DefaultValue $true

if ($GateProfile -eq "stable") {
    # 稳定门禁：以“可复现通过 + 识别明显退化”为目标，降低边界抖动误报。
    $L2MinSuccessRate = [Math]::Max($L2MinSuccessRate, 0.95)
    $L2MaxP95Ms = [Math]::Max($L2MaxP95Ms, 5800)
    $L2MaxP99Ms = [Math]::Max($L2MaxP99Ms, 7000)
    $L3MinSuccessRate = [Math]::Min($L3MinSuccessRate, 0.94)
    $L3MaxP95Ms = [Math]::Max($L3MaxP95Ms, 6200)
    $L3MaxP99Ms = [Math]::Max($L3MaxP99Ms, 8000)
} else {
    # 严格门禁：用于优化阶段压榨尾延迟，发现性能压线问题。
    $L2MinSuccessRate = [Math]::Max($L2MinSuccessRate, 0.95)
    $L2MaxP95Ms = [Math]::Min($L2MaxP95Ms, 5200)
    $L2MaxP99Ms = [Math]::Min($L2MaxP99Ms, 6500)
    $L3MinSuccessRate = [Math]::Max($L3MinSuccessRate, 0.95)
    $L3MaxP95Ms = [Math]::Min($L3MaxP95Ms, 6000)
    $L3MaxP99Ms = [Math]::Min($L3MaxP99Ms, 7800)
}

function Write-Info {
    param([string]$Msg)
    Write-Host "[suite] $Msg"
}

function Assert-Healthy {
    param([string]$Url)
    $resp = Invoke-WebRequest -UseBasicParsing -TimeoutSec 2 "$Url/healthz"
    if ($resp.StatusCode -ne 200) {
        throw "health check failed: $Url => $($resp.StatusCode)"
    }
}

function Assert-PortListening {
    param([int]$Port)
    $listen = Get-NetTCPConnection -State Listen -LocalPort $Port -ErrorAction SilentlyContinue
    if ($null -eq $listen) {
        throw "port not listening: $Port"
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
    Write-Info "token pool is insufficient ($count/$MinCount), regenerating"
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
    $login = {
        $body = @{ username = $Username.Trim(); password = $Password.Trim() } | ConvertTo-Json -Compress
        $resp = Invoke-RestMethod -Method Post -Uri "$($Base.TrimEnd('/'))/api/v1/admin/auth/login" -ContentType "application/json" -Body $body
        if ($resp.code -ne "OK" -or [string]::IsNullOrWhiteSpace($resp.data.access_token)) {
            throw "admin login failed"
        }
        return $resp.data.access_token.Trim()
    }
    $canLogin = (-not [string]::IsNullOrWhiteSpace($Username) -and -not [string]::IsNullOrWhiteSpace($Password))

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

    if ($canLogin -and [string]::IsNullOrWhiteSpace($Token) -and -not (Test-Path $TokenFile)) {
        return (& $login)
    }

    if (-not [string]::IsNullOrWhiteSpace($Token)) {
        $tokenTrimmed = $Token.Trim()
        if ((& $validateToken $tokenTrimmed)) {
            return $tokenTrimmed
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
    if (-not (Test-Path $TokenFile)) {
        throw "admin token file not found: $TokenFile"
    }
    $raw = (Get-Content -Raw $TokenFile).Trim()
    if ([string]::IsNullOrWhiteSpace($raw)) {
        throw "admin token is empty: $TokenFile"
    }
    if ((& $validateToken $raw)) {
        return $raw
    }
    throw "admin token is invalid: $TokenFile"
}

function Get-ActivitySnapshot {
    param(
        [string]$AdminBase,
        [string]$AdminToken,
        [long]$ActivityID
    )
    $headers = @{ Authorization = "Bearer $AdminToken" }
    $detail = Invoke-RestMethod -Method Get -Uri "$($AdminBase.TrimEnd('/'))/api/v1/admin/seckill/activities/$ActivityID" -Headers $headers
    if ($detail.code -ne "OK" -or $null -eq $detail.data.activity) {
        throw "query activity failed: $ActivityID"
    }
    $item = $detail.data.activity.items | Select-Object -First 1
    if ($null -eq $item) {
        throw "activity has no item: $ActivityID"
    }
    return [pscustomobject]@{
        reserved  = [long]$item.reserved_stock_total
        sold      = [long]$item.sold_stock
        available = [long]$item.available_stock
    }
}

function Offline-Activity {
    param(
        [string]$AdminBase,
        [string]$AdminToken,
        [long]$ActivityID
    )
    $headers = @{ Authorization = "Bearer $AdminToken"; "Content-Type" = "application/json" }
    $resp = Invoke-RestMethod -Method Post -Uri "$($AdminBase.TrimEnd('/'))/api/v1/admin/seckill/activities/$ActivityID/offline" -Headers $headers -Body "{}"
    return $resp
}

function Cleanup-StalePerfActivities {
    param(
        [string]$AdminBase,
        [string]$AdminToken
    )
    $headers = @{ Authorization = "Bearer $AdminToken" }
    $page = 1
    $pageSize = 100
    $offlineCount = 0
    while ($true) {
        $resp = Invoke-RestMethod -Method Get -Uri "$($AdminBase.TrimEnd('/'))/api/v1/admin/seckill/activities?page=$page&page_size=$pageSize" -Headers $headers
        if ($resp.code -ne "OK" -or $null -eq $resp.data.list) {
            break
        }
        $list = @($resp.data.list)
        if ($list.Count -eq 0) {
            break
        }
        foreach ($act in $list) {
            $title = ""
            if ($null -ne $act.title) {
                $title = "$($act.title)"
            }
            if (-not $title.StartsWith("perf-activity-")) {
                continue
            }
            [long]$aid = [long]$act.activity_id
            if ($aid -le 0) {
                continue
            }
            try {
                [void](Offline-Activity -AdminBase $AdminBase -AdminToken $AdminToken -ActivityID $aid)
                $offlineCount++
            } catch {
                # Non-blocking cleanup: continue with remaining activities.
            }
        }
        if ($list.Count -lt $pageSize) {
            break
        }
        $page++
        if ($page -gt 20) {
            break
        }
    }
    return $offlineCount
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
    param(
        [string[]]$ToolArgs,
        [string]$RawLogFile
    )
    $lines = @()
    if ($LoadRunnerMode -eq "binary") {
        Write-Info "run binary: $LoadBinaryPath $($ToolArgs -join ' ')"
        & $LoadBinaryPath @ToolArgs 2>&1 | ForEach-Object {
            $lines += "$_"
        }
    } else {
        $goArgs = @("run", "./cmd/perf/seckillload") + $ToolArgs
        Write-Info "run go: go $($goArgs -join ' ')"
        & go @goArgs 2>&1 | ForEach-Object {
            $lines += "$_"
        }
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

function Test-Thresholds {
    param(
        [object]$Summary,
        [hashtable]$Thresholds
    )
    $reasons = New-Object System.Collections.Generic.List[string]
    $pass = $true
    if ($Thresholds.ContainsKey("max_network_error_rate")) {
        if ([double]$Summary.network_error_rate -gt [double]$Thresholds["max_network_error_rate"]) {
            $pass = $false
            $reasons.Add("network_error_rate=$([math]::Round($Summary.network_error_rate,4)) > $($Thresholds["max_network_error_rate"])")
        }
    }
    if ($Thresholds.ContainsKey("min_success_rate")) {
        if ([double]$Summary.success_rate -lt [double]$Thresholds["min_success_rate"]) {
            $pass = $false
            $reasons.Add("success_rate=$([math]::Round($Summary.success_rate,4)) < $($Thresholds["min_success_rate"])")
        }
    }
    if ($Thresholds.ContainsKey("max_p95_ms")) {
        if ([double]$Summary.latency_p95_ms -gt [double]$Thresholds["max_p95_ms"]) {
            $pass = $false
            $reasons.Add("p95=$([math]::Round($Summary.latency_p95_ms,2))ms > $($Thresholds["max_p95_ms"])ms")
        }
    }
    if ($Thresholds.ContainsKey("max_p99_ms")) {
        if ([double]$Summary.latency_p99_ms -gt [double]$Thresholds["max_p99_ms"]) {
            $pass = $false
            $reasons.Add("p99=$([math]::Round($Summary.latency_p99_ms,2))ms > $($Thresholds["max_p99_ms"])ms")
        }
    }
    if ($Thresholds.ContainsKey("max_unique_orders")) {
        if ([int]$Summary.unique_orders -gt [int]$Thresholds["max_unique_orders"]) {
            $pass = $false
            $reasons.Add("unique_orders=$($Summary.unique_orders) > $($Thresholds["max_unique_orders"])")
        }
    }
    if ($Thresholds.ContainsKey("max_http_5xx_rate")) {
        $http5xxRate = Get-Http5xxRate -Summary $Summary
        if ($http5xxRate -gt [double]$Thresholds["max_http_5xx_rate"]) {
            $pass = $false
            $reasons.Add("http_5xx_rate=$([math]::Round($http5xxRate,4)) > $($Thresholds["max_http_5xx_rate"])")
        }
    }
    if ($Thresholds.ContainsKey("max_db_error_rate")) {
        $dbErrorRate = Get-BusinessCodeRate -Summary $Summary -Code "DB_ERROR"
        if ($dbErrorRate -gt [double]$Thresholds["max_db_error_rate"]) {
            $pass = $false
            $reasons.Add("db_error_rate=$([math]::Round($dbErrorRate,4)) > $($Thresholds["max_db_error_rate"])")
        }
    }
    if ($Thresholds.ContainsKey("min_track_accept_rate")) {
        if ([double]$Summary.track_accept_rate -lt [double]$Thresholds["min_track_accept_rate"]) {
            $pass = $false
            $reasons.Add("track_accept_rate=$([math]::Round($Summary.track_accept_rate,4)) < $($Thresholds["min_track_accept_rate"])")
        }
    }
    return [pscustomobject]@{
        pass    = $pass
        reasons = $reasons
    }
}

function Get-Http5xxRate {
    param([object]$Summary)
    if ($null -eq $Summary -or [int]$Summary.total -le 0 -or $null -eq $Summary.http_status) {
        return 0.0
    }
    $sum = 0
    foreach ($entry in $Summary.http_status.PSObject.Properties) {
        $code = 0
        if ([int]::TryParse([string]$entry.Name, [ref]$code) -and $code -ge 500 -and $code -le 599) {
            $sum += [int]$entry.Value
        }
    }
    return ([double]$sum / [double]$Summary.total)
}

function Get-BusinessCodeRate {
    param(
        [object]$Summary,
        [string]$Code
    )
    if ($null -eq $Summary -or [int]$Summary.total -le 0 -or $null -eq $Summary.business_code -or [string]::IsNullOrWhiteSpace($Code)) {
        return 0.0
    }
    foreach ($entry in $Summary.business_code.PSObject.Properties) {
        if ([string]::Equals($entry.Name, $Code, [System.StringComparison]::OrdinalIgnoreCase)) {
            return ([double]$entry.Value / [double]$Summary.total)
        }
    }
    return 0.0
}

$timestamp = Get-Date -Format "yyyyMMdd-HHmmss"
$OutputDir = Join-Path $OutputRoot "suite-$timestamp"
New-Item -ItemType Directory -Path $OutputDir -Force | Out-Null

Write-Info "L0 precheck"
Write-Info "runtime options: GateProfile=$GateProfile EnableL3=$EnableL3 StopOnFirstFailure=$stopOnFirstFailureEnabled AutoDiagnoseOnL2Failure=$autoDiagnoseOnL2FailureEnabled LoadRunnerMode=$LoadRunnerMode Stabilize=$EnableStageStabilize L2(c=$L2PurchaseConcurrency,r=$L2PurchaseRequests) L3(c=$L3PurchaseConcurrency,r=$L3PurchaseRequests)"
if ($RestartRuntimeBeforeRun) {
    Write-Info "restart runtime before suite: ipv4=$RuntimeForceIPv4Loopback mysql_open=$RuntimeMySQLMaxOpenConns mysql_idle=$RuntimeMySQLMaxIdleConns reserve_timeout_ms=$RuntimeReservePurchaseTimeoutMs"
    & (Join-Path $PSScriptRoot "restart-seckill-runtime.ps1") `
        -RestartOrderRPC:$true `
        -RestartSeckillRPC:$true `
        -RestartUserGateway:$false `
        -EnvFile $EnvFile `
        -ForceIPv4Loopback:$RuntimeForceIPv4Loopback `
        -MySQLMaxOpenConns $RuntimeMySQLMaxOpenConns `
        -MySQLMaxIdleConns $RuntimeMySQLMaxIdleConns `
        -SeckillReservePurchaseTimeoutMs $RuntimeReservePurchaseTimeoutMs
}
Assert-Healthy -Url $BaseUrl
Assert-Healthy -Url $AdminBaseUrl
foreach ($p in 8084, 8085, 8086, 8087) {
    Assert-PortListening -Port $p
}
$tokenCount = Ensure-TokenPool -Path $TokenFile -MinCount $MinTokenCount
$adminAccessToken = Resolve-AdminToken -Base $AdminBaseUrl -Token $AdminToken -TokenFile $AdminTokenFile -Username $AdminUsername -Password $AdminPassword
Write-Info "token pool ready: $tokenCount"
if ($LoadRunnerMode -eq "binary") {
    if ($BuildLoadBinary -or -not (Test-Path $LoadBinaryPath)) {
        Write-Info "building seckillload binary"
        Ensure-SeckillLoadBinary -Path $LoadBinaryPath
    }
    Write-Info "seckillload binary ready: $LoadBinaryPath"
}

if ($CleanupStalePerfActivities) {
    $cleaned = Cleanup-StalePerfActivities -AdminBase $AdminBaseUrl -AdminToken $adminAccessToken
    Write-Info "stale perf activities cleaned: $cleaned"
}

if ($Prepare) {
    Write-Info "prepare fresh activity/item"
    $candidates = @($ReservedStockTotal, 12000, 8000, 5000) | Where-Object { $_ -gt 0 } | Select-Object -Unique
    $prepared = $false
    $lastPrepareError = $null
    foreach ($stock in $candidates) {
        try {
            $prepareOutput = powershell -ExecutionPolicy Bypass -File scripts/dev/seckill-prepare.ps1 `
                -BaseUrl $AdminBaseUrl `
                -AdminToken $adminAccessToken `
                -ProductId $ProductId `
                -ProductIdFile $ProductIdFile `
                -DurationMinutes $DurationMinutes `
                -ReservedStockTotal $stock `
                -UserLimitQty $UserLimitQty 2>&1
            $prepareExit = $LASTEXITCODE
            if ($prepareExit -ne 0) {
                throw ("prepare exited with code {0}: {1}" -f $prepareExit, $prepareOutput)
            }
            [long]$newActivityID = (Get-Content -Raw .memory/runlogs/perf.activity_id.txt).Trim()
            $snap = Get-ActivitySnapshot -AdminBase $AdminBaseUrl -AdminToken $adminAccessToken -ActivityID $newActivityID
            if ($snap.sold -gt 0) {
                throw "prepared activity is not fresh (sold_stock=$($snap.sold))"
            }
            Write-Info "prepare succeeded with reserved_stock_total=$stock activity_id=$newActivityID"
            $prepared = $true
            break
        } catch {
            $lastPrepareError = $_
            if ($_.Exception.Message -match "SECKILL_OUT_OF_STOCK|out of stock") {
                Write-Info "prepare stock=$stock out of stock, retry with smaller stock"
                continue
            }
            throw
        }
    }
    if (-not $prepared) {
        throw "prepare failed after fallback: $lastPrepareError"
    }
}

[long]$activityID = (Get-Content -Raw .memory/runlogs/perf.activity_id.txt).Trim()
[long]$itemID = (Get-Content -Raw .memory/runlogs/perf.item_id.txt).Trim()
$singleToken = (Get-Content $TokenFile | Where-Object { -not [string]::IsNullOrWhiteSpace($_) } | Select-Object -First 1).Trim()
if ($activityID -le 0 -or $itemID -le 0) {
    throw "invalid activity/item id"
}

$stages = New-Object System.Collections.Generic.List[object]

function Run-Stage {
    param(
        [string]$Name,
        [string]$Scenario,
        [int]$Concurrency,
        [int]$Requests,
        [int]$TimeoutMs,
        [int]$MaxConnsPerHost,
        [hashtable]$Thresholds,
        [switch]$UseSingleToken,
        [bool]$Gate = $true
    )
    Write-Info "running $Name ($Scenario c=$Concurrency r=$Requests t=${TimeoutMs}ms)"
    Write-Info "stage ids: activity=$activityID item=$itemID"
    $args = @(
        "-scenario", $Scenario,
        "-base-url", $BaseUrl,
        "-activity-id", "$activityID",
        "-item-id", "$itemID",
        "-concurrency", "$Concurrency",
        "-requests", "$Requests",
        "-timeout", "$($TimeoutMs)ms",
        "-max-idle-conns", "1024",
        "-max-idle-conns-per-host", "512",
        "-max-conns-per-host", "$MaxConnsPerHost",
        "-output", "json",
        "-max-network-errors", "999999"
    )
    if ($Scenario -eq "track-stress") {
        $args += @("-event-type", "pv")
    } elseif ($UseSingleToken) {
        $args += @("-token", $singleToken)
    } else {
        $args += @("-token-file", $TokenFile)
    }
    $rawLog = Join-Path $OutputDir "$Name.raw.log"
    $json = Invoke-SeckillLoadJson -ToolArgs $args -RawLogFile $rawLog
    $check = Test-Thresholds -Summary $json.summary -Thresholds $Thresholds
    $http5xxRate = Get-Http5xxRate -Summary $json.summary
    $dbErrorRate = Get-BusinessCodeRate -Summary $json.summary -Code "DB_ERROR"
    $stage = [pscustomobject]@{
        name               = $Name
        scenario           = $Scenario
        gate               = $Gate
        pass               = $check.pass
        reasons            = ($check.reasons -join "; ")
        requests           = $json.config.requests
        concurrency        = $json.config.concurrency
        success_rate       = [double]$json.summary.success_rate
        network_error_rate = [double]$json.summary.network_error_rate
        p95_ms             = [double]$json.summary.latency_p95_ms
        p99_ms             = [double]$json.summary.latency_p99_ms
        rps                = [double]$json.summary.rps
        http_5xx_rate      = [double]$http5xxRate
        db_error_rate      = [double]$dbErrorRate
        unique_orders      = [int]$json.summary.unique_orders
        track_accept_rate  = [double]$json.summary.track_accept_rate
        raw_log            = $rawLog
        json               = $json
    }
    $stages.Add($stage) | Out-Null
    $jsonPath = Join-Path $OutputDir "$Name.json"
    ($json | ConvertTo-Json -Depth 10) | Set-Content $jsonPath
    if ($stage.pass) {
        Write-Info "$Name PASS"
    } else {
        Write-Info "$Name FAIL: $($stage.reasons)"
    }
    return $stage
}

function Wait-StageCooldown {
    param(
        [string]$AfterStage,
        [int]$Seconds
    )
    if ($Seconds -le 0) {
        return
    }
    Write-Info "cooldown after ${AfterStage}: ${Seconds}s"
    Start-Sleep -Seconds $Seconds
}

function Wait-StageStabilize {
    param(
        [string]$AfterStage
    )
    if (-not $EnableStageStabilize) {
        return
    }
    if ($StageStabilizeMaxWaitSeconds -le 0) {
        return
    }
    $start = Get-Date
    $safeStage = ($AfterStage -replace "[^a-zA-Z0-9\\-]", "_")
    while ($true) {
        $elapsed = [int]((Get-Date) - $start).TotalSeconds
        if ($elapsed -ge $StageStabilizeMaxWaitSeconds) {
            Write-Info "stabilize timeout after $AfterStage (${elapsed}s), continue"
            return
        }
        $probeArgs = @(
            "-scenario", "purchase-stress",
            "-base-url", $BaseUrl,
            "-activity-id", "$activityID",
            "-item-id", "$itemID",
            "-concurrency", "$StageStabilizeProbeConcurrency",
            "-requests", "$StageStabilizeProbeRequests",
            "-timeout", "4000ms",
            "-max-idle-conns", "256",
            "-max-idle-conns-per-host", "128",
            "-max-conns-per-host", "64",
            "-output", "json",
            "-max-network-errors", "999999",
            "-token-file", $TokenFile
        )
        $probeRaw = Join-Path $OutputDir "stabilize-$safeStage-${elapsed}s.raw.log"
        try {
            $probe = Invoke-SeckillLoadJson -ToolArgs $probeArgs -RawLogFile $probeRaw
            $ok = ([double]$probe.summary.success_rate -ge $StageStabilizeMinSuccessRate -and [double]$probe.summary.network_error_rate -le 0.02)
            if ($ok) {
                Write-Info "stabilized after $AfterStage (${elapsed}s): success_rate=$([math]::Round($probe.summary.success_rate,4))"
                return
            }
            Write-Info "stabilizing after ${AfterStage}: success_rate=$([math]::Round($probe.summary.success_rate,4)), network_error_rate=$([math]::Round($probe.summary.network_error_rate,4))"
        } catch {
            Write-Info "stabilize probe error after ${AfterStage}: $($_.Exception.Message)"
        }
        Start-Sleep -Seconds 5
    }
}

$stageIdem = Run-Stage -Name "L1-idempotency" -Scenario "idempotency" -Concurrency 80 -Requests 600 -TimeoutMs 5000 -MaxConnsPerHost 80 -UseSingleToken -Thresholds @{
    max_unique_orders     = 1
    max_network_error_rate = 0.01
    max_http_5xx_rate     = 0.005
    max_db_error_rate     = 0
    max_p95_ms            = 1500
}
if ($stopOnFirstFailureEnabled -and -not $stageIdem.pass) {
    Write-Info "stop on first failure at L1"
}
Wait-StageCooldown -AfterStage $stageIdem.name -Seconds $StageCooldownSeconds
Wait-StageStabilize -AfterStage $stageIdem.name

$purchaseProceed = -not ($stopOnFirstFailureEnabled -and -not $stageIdem.pass)
if ($purchaseProceed) {
    $stageP100 = Run-Stage -Name "L2-purchase-c100" -Scenario "purchase-stress" -Concurrency $L2C100Concurrency -Requests $L2C100Requests -TimeoutMs $L2C100TimeoutMs -MaxConnsPerHost $L2C100MaxConnsPerHost -Thresholds @{
        max_network_error_rate = $L2C100MaxNetworkErrorRate
        max_http_5xx_rate      = [Math]::Min([double]$L2C100MaxNetworkErrorRate, 0.08)
        min_success_rate       = $L2C100MinSuccessRate
        max_p95_ms             = $L2C100MaxP95Ms
        max_p99_ms             = $L2C100MaxP99Ms
    }
    if (-not $stageP100.pass -and $stopOnFirstFailureEnabled) {
        $purchaseProceed = $false
    }
    Wait-StageCooldown -AfterStage $stageP100.name -Seconds $StageCooldownSeconds
    Wait-StageStabilize -AfterStage $stageP100.name
}
if ($purchaseProceed) {
    $stageP200Name = "L2-purchase-c$L2PurchaseConcurrency"
    $stageP200 = Run-Stage -Name $stageP200Name -Scenario "purchase-stress" -Concurrency $L2PurchaseConcurrency -Requests $L2PurchaseRequests -TimeoutMs $L2PurchaseTimeoutMs -MaxConnsPerHost $L2PurchaseMaxConnsPerHost -Thresholds @{
        max_network_error_rate = $L2MaxNetworkErrorRate
        max_http_5xx_rate      = [Math]::Min([double]$L2MaxNetworkErrorRate, 0.08)
        min_success_rate       = $L2MinSuccessRate
        max_p95_ms             = $L2MaxP95Ms
        max_p99_ms             = $L2MaxP99Ms
    }
    if (-not $stageP200.pass -and $stopOnFirstFailureEnabled) {
        $purchaseProceed = $false
    }
    Wait-StageCooldown -AfterStage $stageP200.name -Seconds $StageCooldownSeconds
    Wait-StageStabilize -AfterStage $stageP200.name
    if (-not $stageP200.pass -and $autoDiagnoseOnL2FailureEnabled) {
        $stageP200Diag = Run-Stage -Name "L2b-purchase-c$L2PurchaseConcurrency-diagnose" -Scenario "purchase-stress" -Concurrency $L2PurchaseConcurrency -Requests $L2PurchaseRequests -TimeoutMs ([Math]::Max($L2PurchaseTimeoutMs, 8000)) -MaxConnsPerHost ([Math]::Max($L2PurchaseMaxConnsPerHost, 260)) -Gate $false -Thresholds @{
            max_network_error_rate = 0.05
            max_http_5xx_rate      = 0.08
            min_success_rate       = 0.95
            max_p95_ms             = 7000
            max_p99_ms             = 8500
        }
        Wait-StageCooldown -AfterStage $stageP200Diag.name -Seconds $StageCooldownSeconds
        Wait-StageStabilize -AfterStage $stageP200Diag.name
    }
}
if ($purchaseProceed -and $EnableL3) {
    $stageP300 = Run-Stage -Name "L3-purchase-c$L3PurchaseConcurrency" -Scenario "purchase-stress" -Concurrency $L3PurchaseConcurrency -Requests $L3PurchaseRequests -TimeoutMs $L3PurchaseTimeoutMs -MaxConnsPerHost $L3PurchaseMaxConnsPerHost -Thresholds @{
        max_network_error_rate = $L3MaxNetworkErrorRate
        max_http_5xx_rate      = [Math]::Min([double]$L3MaxNetworkErrorRate, 0.12)
        min_success_rate       = $L3MinSuccessRate
        max_p95_ms             = $L3MaxP95Ms
        max_p99_ms             = $L3MaxP99Ms
    }
    Wait-StageCooldown -AfterStage $stageP300.name -Seconds $StageCooldownSeconds
    Wait-StageStabilize -AfterStage $stageP300.name
}

[void](Run-Stage -Name "L4-track-c300" -Scenario "track-stress" -Concurrency 300 -Requests 6000 -TimeoutMs $TrackTimeoutMs -MaxConnsPerHost $TrackMaxConnsPerHost -Thresholds @{
    max_network_error_rate = 0.01
    max_http_5xx_rate      = 0.01
    min_success_rate       = 0.99
    min_track_accept_rate  = 0.99
    max_p95_ms             = 500
})

Write-Info "L5 reconciliation"
$recon = [pscustomobject]@{
    activity_id      = $activityID
    item_id          = $itemID
    reserved_stock   = 0
    sold_stock       = 0
    available_stock  = 0
    orders_total     = 0
    reconcile_status = "unknown"
}
try {
    $headers = @{ Authorization = "Bearer $adminAccessToken" }
    $detail = Invoke-RestMethod -Method Get -Uri "$($AdminBaseUrl.TrimEnd('/'))/api/v1/admin/seckill/activities/$activityID" -Headers $headers
    $item = $detail.data.activity.items | Select-Object -First 1
    $orders = Invoke-RestMethod -Method Get -Uri "$($AdminBaseUrl.TrimEnd('/'))/api/v1/admin/seckill/activities/$activityID/orders?page=1&page_size=1" -Headers $headers
    $recon.reserved_stock = [long]$item.reserved_stock_total
    $recon.sold_stock = [long]$item.sold_stock
    $recon.available_stock = [long]$item.available_stock
    $recon.orders_total = [long]$orders.data.total
    $sum = $recon.sold_stock + $recon.available_stock
    if ($sum -le $recon.reserved_stock) {
        $recon.reconcile_status = "pass"
    } else {
        $recon.reconcile_status = "fail"
    }
} catch {
    $recon.reconcile_status = "error: $($_.Exception.Message)"
}

$failedCount = ($stages | Where-Object { $_.gate -and -not $_.pass }).Count
$overallPass = $failedCount -eq 0 -and $recon.reconcile_status -eq "pass"

if ($CleanupCurrentActivity) {
    try {
        [void](Offline-Activity -AdminBase $AdminBaseUrl -AdminToken $adminAccessToken -ActivityID $activityID)
        Write-Info "current activity offlined: $activityID"
    } catch {
        Write-Info "current activity offline failed: $($_.Exception.Message)"
    }
}

$reportPath = Join-Path $OutputDir "suite-report.md"
$sb = New-Object System.Text.StringBuilder
[void]$sb.AppendLine("# 秒杀压测分层报告")
[void]$sb.AppendLine("")
[void]$sb.AppendLine("- 生成时间: $(Get-Date -Format 'yyyy-MM-dd HH:mm:ss')")
[void]$sb.AppendLine("- BaseUrl: $BaseUrl")
[void]$sb.AppendLine("- Activity/Item: $activityID / $itemID")
[void]$sb.AppendLine("- 结果: $(if($overallPass){'PASS'}else{'FAIL'})")
[void]$sb.AppendLine("")
[void]$sb.AppendLine("## 分层结果")
[void]$sb.AppendLine("")
[void]$sb.AppendLine("| Stage | Gate | Pass | SuccessRate | NetworkErrRate | HTTP5xxRate | DBErrorRate | P95(ms) | P99(ms) | RPS | Notes |")
[void]$sb.AppendLine("|---|---|---|---:|---:|---:|---:|---:|---:|---:|---|")
foreach ($s in $stages) {
    $passText = if ($s.pass) { "PASS" } else { "FAIL" }
    $gateText = if ($s.gate) { "Y" } else { "N" }
    $notes = if ([string]::IsNullOrWhiteSpace($s.reasons)) { "-" } else { $s.reasons }
    [void]$sb.AppendLine("| $($s.name) | $gateText | $passText | $([math]::Round($s.success_rate*100,2))% | $([math]::Round($s.network_error_rate*100,2))% | $([math]::Round($s.http_5xx_rate*100,2))% | $([math]::Round($s.db_error_rate*100,2))% | $([math]::Round($s.p95_ms,2)) | $([math]::Round($s.p99_ms,2)) | $([math]::Round($s.rps,2)) | $notes |")
}
[void]$sb.AppendLine("")
[void]$sb.AppendLine("## 对账")
[void]$sb.AppendLine("")
[void]$sb.AppendLine("- reserved_stock: $($recon.reserved_stock)")
[void]$sb.AppendLine("- sold_stock: $($recon.sold_stock)")
[void]$sb.AppendLine("- available_stock: $($recon.available_stock)")
[void]$sb.AppendLine("- sold+available: $($recon.sold_stock + $recon.available_stock)")
[void]$sb.AppendLine("- orders_total: $($recon.orders_total)")
[void]$sb.AppendLine("- reconcile_status: $($recon.reconcile_status)")
[void]$sb.AppendLine("")
[void]$sb.AppendLine("## 原始输出")
foreach ($s in $stages) {
    [void]$sb.AppendLine("- $($s.name): $($s.raw_log)")
}
[void]$sb.AppendLine("- report: $reportPath")
Set-Content -Path $reportPath -Value $sb.ToString()

Write-Info "report generated: $reportPath"
if (-not $overallPass) {
    exit 2
}
exit 0
