param(
    [ValidateSet("purchase-stress", "idempotency", "track-stress")]
    [string]$Scenario = "purchase-stress",
    [string]$BaseUrl = "http://127.0.0.1:8082",
    [long]$ActivityId,
    [long]$ItemId,
    [int]$Concurrency = 100,
    [int]$Requests = 1000,
    [long]$Quantity = 1,
    [string]$EventType = "pv",
    [string]$ClientIdPrefix = "perf-client",
    [string]$IdempotencyGroup = "same-key-group",
    [int]$ExpectMaxSuccess = -1,
    [int]$MaxNetworkErrors = 999999,
    [int]$TimeoutMs = 3000,
    [int]$MaxIdleConns = 1024,
    [int]$MaxIdleConnsPerHost = 512,
    [int]$MaxConnsPerHost = 0,
    [switch]$DisableKeepAlive,
    [ValidateSet("go-run", "binary")]
    [string]$RunnerMode = "go-run",
    [string]$BinaryPath = ".memory/runlogs/bin/seckillload.exe",
    [switch]$BuildBinary,
    [ValidateSet("text", "json")]
    [string]$Output = "text",
    [string]$Token = "",
    [string]$TokenFile = ""
)

$ErrorActionPreference = "Stop"

if ($ActivityId -le 0 -or $ItemId -le 0) {
    throw "ActivityId and ItemId must be greater than 0."
}

function Ensure-RunnerBinary {
    param(
        [string]$Path
    )
    $dir = Split-Path -Parent $Path
    if (-not [string]::IsNullOrWhiteSpace($dir) -and -not (Test-Path $dir)) {
        New-Item -ItemType Directory -Path $dir -Force | Out-Null
    }
    go build -o $Path ./cmd/perf/seckillload
    if ($LASTEXITCODE -ne 0) {
        throw "build seckillload failed"
    }
}

if ($RunnerMode -eq "binary") {
    if ($BuildBinary -or -not (Test-Path $BinaryPath)) {
        Ensure-RunnerBinary -Path $BinaryPath
    }
}

$timeout = "{0}ms" -f $TimeoutMs
$args = @(
    "run", "./cmd/perf/seckillload",
    "-scenario", $Scenario,
    "-base-url", $BaseUrl,
    "-activity-id", "$ActivityId",
    "-item-id", "$ItemId",
    "-concurrency", "$Concurrency",
    "-requests", "$Requests",
    "-quantity", "$Quantity",
    "-event-type", $EventType,
    "-client-id-prefix", $ClientIdPrefix,
    "-idempotency-group", $IdempotencyGroup,
    "-expect-max-success", "$ExpectMaxSuccess",
    "-max-network-errors", "$MaxNetworkErrors",
    "-timeout", $timeout,
    "-max-idle-conns", "$MaxIdleConns",
    "-max-idle-conns-per-host", "$MaxIdleConnsPerHost",
    "-max-conns-per-host", "$MaxConnsPerHost",
    "-output", $Output
)

if ($DisableKeepAlive) {
    $args += @("-disable-keepalive")
}
if (-not [string]::IsNullOrWhiteSpace($Token)) {
    $args += @("-token", $Token)
}
if (-not [string]::IsNullOrWhiteSpace($TokenFile)) {
    $args += @("-token-file", $TokenFile)
}

if ($RunnerMode -eq "binary") {
    $runnerArgs = @()
    if ($args.Length -gt 2) {
        $runnerArgs = @($args[2..($args.Length - 1)])
    }
    Write-Host ("[seckill-perf] running: {0} {1}" -f $BinaryPath, ($runnerArgs -join " "))
    & $BinaryPath @runnerArgs
    if ($LASTEXITCODE -ne 0) {
        exit $LASTEXITCODE
    }
} else {
    Write-Host ("[seckill-perf] running: go {0}" -f ($args -join " "))
    go @args
    if ($LASTEXITCODE -ne 0) {
        exit $LASTEXITCODE
    }
}
