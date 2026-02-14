param(
    [switch]$Observability
)

$ErrorActionPreference = "Stop"
$repoRoot = (Resolve-Path (Join-Path $PSScriptRoot "..\..\..\")).Path
$devEnvFile = Join-Path $repoRoot "configs\local\dev.env"
$composeFile = Join-Path $repoRoot "deploy\compose\docker-compose.yml"

. (Join-Path $PSScriptRoot "..\common.ps1")
Load-DevEnv -Path $devEnvFile

$cmd = @("compose", "--env-file", $devEnvFile, "-f", $composeFile)
if ($Observability) {
    $cmd += @("--profile", "observability")
}
$cmd += @("up", "-d")

Write-Host "Starting containers..."
& docker @cmd
if ($LASTEXITCODE -ne 0) {
    throw "docker compose up failed with exit code $LASTEXITCODE"
}

Write-Host "Containers are starting. Run scripts/dev/env/smoke.ps1 after healthy."


