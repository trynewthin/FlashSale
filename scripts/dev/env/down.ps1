param(
    [switch]$RemoveVolumes
)

$ErrorActionPreference = "Stop"
$repoRoot = (Resolve-Path (Join-Path $PSScriptRoot "..\..\..\")).Path
$devEnvFile = Join-Path $repoRoot "configs\local\dev.env"
$composeFile = Join-Path $repoRoot "deploy\compose\docker-compose.yml"

. (Join-Path $PSScriptRoot "..\common.ps1")
Load-DevEnv -Path $devEnvFile

$cmd = @("compose", "--env-file", $devEnvFile, "-f", $composeFile, "down")
if ($RemoveVolumes) {
    $cmd += "-v"
}

& docker @cmd
if ($LASTEXITCODE -ne 0) {
    throw "docker compose down failed with exit code $LASTEXITCODE"
}
Write-Host "Containers stopped."

