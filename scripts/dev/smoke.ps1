$ErrorActionPreference = "Stop"
$repoRoot = (Resolve-Path (Join-Path $PSScriptRoot "..\..\")).Path
$devEnvFile = Join-Path $repoRoot "configs\local\dev.env"

. (Join-Path $PSScriptRoot "common.ps1")
Load-DevEnv -Path $devEnvFile

Push-Location $repoRoot
try {
    Write-Host "Running mysql smoke check"
    & go run ./cmd/smoke/mysqlcheck -config configs/local/dev.yaml
    if ($LASTEXITCODE -ne 0) { throw "mysql smoke check failed with exit code $LASTEXITCODE" }

    Write-Host "Running redis smoke check"
    & go run ./cmd/smoke/redischeck -config configs/local/dev.yaml
    if ($LASTEXITCODE -ne 0) { throw "redis smoke check failed with exit code $LASTEXITCODE" }

    Write-Host "Running kafka smoke check"
    & go run ./cmd/smoke/kafkacheck -config configs/local/dev.yaml
    if ($LASTEXITCODE -ne 0) { throw "kafka smoke check failed with exit code $LASTEXITCODE" }

    Write-Host "All smoke checks passed"
} finally {
    Pop-Location
}
