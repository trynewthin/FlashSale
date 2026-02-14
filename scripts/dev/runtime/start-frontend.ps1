param(
    [switch]$InstallDeps,
    [switch]$KillExisting = $true,
    [int]$UserPort = 5173,
    [int]$AdminPort = 5174
)

$ErrorActionPreference = "Stop"
$repoRoot = (Resolve-Path (Join-Path $PSScriptRoot "..\..\..\")).Path
$logDir = Join-Path $repoRoot ".memory/runlogs/frontends"
New-Item -ItemType Directory -Force -Path $logDir | Out-Null

function Resolve-Runner {
    $bun = Get-Command bun -ErrorAction SilentlyContinue
    if ($bun) {
        return [PSCustomObject]@{
            Command    = "bun"
            InstallArg = @("install")
            DevPrefix  = @("run", "dev", "--")
        }
    }

    $npm = Get-Command npm -ErrorAction SilentlyContinue
    if ($npm) {
        return [PSCustomObject]@{
            Command    = "npm"
            InstallArg = @("install")
            DevPrefix  = @("run", "dev", "--")
        }
    }

    throw "bun/npm not found"
}

function Stop-PortProcess {
    param([int]$Port)
    $listen = Get-NetTCPConnection -State Listen -LocalPort $Port -ErrorAction SilentlyContinue | Select-Object -First 1
    if ($listen) {
        Stop-Process -Id $listen.OwningProcess -Force -ErrorAction SilentlyContinue
        Start-Sleep -Milliseconds 500
    }
}

function Wait-PortReady {
    param([int]$Port, [int]$TimeoutSec = 90)
    $deadline = (Get-Date).AddSeconds($TimeoutSec)
    while ((Get-Date) -lt $deadline) {
        if (Get-NetTCPConnection -State Listen -LocalPort $Port -ErrorAction SilentlyContinue) {
            return $true
        }
        Start-Sleep -Milliseconds 400
    }
    return $false
}

function Get-ListenPid {
    param([int]$Port)
    $listen = Get-NetTCPConnection -State Listen -LocalPort $Port -ErrorAction SilentlyContinue | Select-Object -First 1
    if ($listen) {
        return [int]$listen.OwningProcess
    }
    return 0
}

function Start-FrontendApp {
    param(
        [string]$Name,
        [string]$Dir,
        [int]$Port,
        [object]$Runner,
        [bool]$DoInstall
    )
    $appDir = Join-Path $repoRoot $Dir
    if (-not (Test-Path $appDir)) {
        throw "frontend dir not found: $Dir"
    }

    if ($DoInstall) {
        Write-Host "[start-frontend] install deps: $Name ($($Runner.Command))"
        Push-Location $appDir
        try {
            & $Runner.Command @($Runner.InstallArg)
            if ($LASTEXITCODE -ne 0) {
                throw "install deps failed for $Name"
            }
        } finally {
            Pop-Location
        }
    }

    $stdoutPath = Join-Path $logDir "$Name.stdout.log"
    $stderrPath = Join-Path $logDir "$Name.stderr.log"
    if (Test-Path $stdoutPath) { Remove-Item $stdoutPath -Force }
    if (Test-Path $stderrPath) { Remove-Item $stderrPath -Force }

    $args = @()
    $args += $Runner.DevPrefix
    $args += @("--host", "0.0.0.0", "--port", "$Port")

    $proc = Start-Process -FilePath $Runner.Command `
        -ArgumentList $args `
        -WorkingDirectory $appDir `
        -PassThru `
        -WindowStyle Hidden `
        -RedirectStandardOutput $stdoutPath `
        -RedirectStandardError $stderrPath

    if (-not (Wait-PortReady -Port $Port -TimeoutSec 90)) {
        throw "$Name not ready on port $Port. check logs in $logDir"
    }
    return $proc
}

$runner = Resolve-Runner
Write-Host "[start-frontend] runner = $($runner.Command)"

if ($KillExisting) {
    Stop-PortProcess -Port $UserPort
    Stop-PortProcess -Port $AdminPort
}

$userProc = Start-FrontendApp -Name "frontend-user" -Dir "frontend/user" -Port $UserPort -Runner $runner -DoInstall:$InstallDeps
$adminProc = Start-FrontendApp -Name "frontend-admin" -Dir "frontend/admin" -Port $AdminPort -Runner $runner -DoInstall:$InstallDeps
$userListenPid = Get-ListenPid -Port $UserPort
$adminListenPid = Get-ListenPid -Port $AdminPort

$pidFile = Join-Path $logDir "frontend.pids.json"
@(
    @{ Name = "frontend-user"; Port = $UserPort; ListenPID = $userListenPid; LauncherPID = $userProc.Id; URL = "http://127.0.0.1:$UserPort" },
    @{ Name = "frontend-admin"; Port = $AdminPort; ListenPID = $adminListenPid; LauncherPID = $adminProc.Id; URL = "http://127.0.0.1:$AdminPort" }
) | ConvertTo-Json -Depth 4 | Set-Content -Path $pidFile -Encoding utf8

Write-Host "[start-frontend] ready"
Write-Host "  - user : http://127.0.0.1:$UserPort (listen_pid=$userListenPid launcher_pid=$($userProc.Id))"
Write-Host "  - admin: http://127.0.0.1:$AdminPort (listen_pid=$adminListenPid launcher_pid=$($adminProc.Id))"
Write-Host "[start-frontend] pid file: $pidFile"

