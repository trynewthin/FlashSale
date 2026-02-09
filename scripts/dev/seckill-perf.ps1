param(
    [ValidateSet("purchase-stress", "idempotency", "track-stress")]
    [string]$Scenario = "purchase-stress",
    [string]$BaseUrl = "http://127.0.0.1:8081",
    [long]$ActivityId,
    [long]$ItemId,
    [int]$Concurrency = 100,
    [int]$Requests = 1000,
    [long]$Quantity = 1,
    [string]$EventType = "pv",
    [string]$ClientIdPrefix = "perf-client",
    [string]$IdempotencyGroup = "same-key-group",
    [int]$ExpectMaxSuccess = -1,
    [int]$MaxNetworkErrors = 0,
    [int]$TimeoutMs = 3000,
    [string]$Token = "",
    [string]$TokenFile = ""
)

$ErrorActionPreference = "Stop"

if ($ActivityId -le 0 -or $ItemId -le 0) {
    throw "ActivityId 和 ItemId 必须大于 0。"
}

$timeout = "$($TimeoutMs)ms"
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
    "-timeout", $timeout
)

if (-not [string]::IsNullOrWhiteSpace($Token)) {
    $args += @("-token", $Token)
}
if (-not [string]::IsNullOrWhiteSpace($TokenFile)) {
    $args += @("-token-file", $TokenFile)
}

Write-Host "[seckill-perf] running: go $($args -join ' ')"
go @args
if ($LASTEXITCODE -ne 0) {
    exit $LASTEXITCODE
}
