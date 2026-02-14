param(
    [switch]$Observability,
    [switch]$SkipMigrate,
    [switch]$SkipSmoke
)

$ErrorActionPreference = "Stop"
$repoRoot = (Resolve-Path (Join-Path $PSScriptRoot "..\..\..\")).Path

function Invoke-Step {
    param(
        [string]$Name,
        [string]$ScriptPath,
        [string[]]$ExtraArgs
    )
    Write-Host "[start-docker-env] $Name"
    & $ScriptPath @ExtraArgs
    if ($LASTEXITCODE -ne 0) {
        throw "$Name failed with exit code $LASTEXITCODE"
    }
}

Push-Location $repoRoot
try {
    $upScript = Join-Path $repoRoot "scripts/dev/env/up.ps1"
    $upArgs = @()
    if ($Observability) {
        $upArgs += "-Observability"
    }
    Invoke-Step -Name "docker compose up" -ScriptPath $upScript -ExtraArgs $upArgs

    if (-not $SkipMigrate) {
        Invoke-Step -Name "migrate up" -ScriptPath (Join-Path $repoRoot "scripts/dev/env/migrate-up.ps1") -ExtraArgs @()
    } else {
        Write-Host "[start-docker-env] skip migrate"
    }

    if (-not $SkipSmoke) {
        Invoke-Step -Name "smoke checks" -ScriptPath (Join-Path $repoRoot "scripts/dev/env/smoke.ps1") -ExtraArgs @()
    } else {
        Write-Host "[start-docker-env] skip smoke"
    }

    Write-Host "[start-docker-env] done"
} finally {
    Pop-Location
}


