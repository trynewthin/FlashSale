param(
    [int[]]$Ports = @(5173, 5174),
    [string]$PidFile = ".memory/runlogs/frontends/frontend.pids.json"
)

$ErrorActionPreference = "Stop"
$repoRoot = (Resolve-Path (Join-Path $PSScriptRoot "..\..\..\")).Path
$pidFilePath = Join-Path $repoRoot $PidFile

function Stop-ByPid {
    param([int]$TargetPid)
    if ($TargetPid -le 0) {
        return
    }
    if ($TargetPid -eq $PID) {
        return
    }
    try {
        $proc = Get-Process -Id $TargetPid -ErrorAction SilentlyContinue
        if ($proc) {
            Stop-Process -Id $TargetPid -Force -ErrorAction SilentlyContinue
            Start-Sleep -Milliseconds 250
        }
    } catch {
    }
}

function Stop-ByPort {
    param([int]$Port)
    $listen = Get-NetTCPConnection -State Listen -LocalPort $Port -ErrorAction SilentlyContinue | Select-Object -First 1
    if ($listen) {
        Stop-ByPid -TargetPid $listen.OwningProcess
    }
}

$targetPids = New-Object System.Collections.Generic.HashSet[int]
if (Test-Path $pidFilePath) {
    try {
        $data = Get-Content $pidFilePath -Raw | ConvertFrom-Json
        foreach ($item in @($data)) {
            if ($null -ne $item.ListenPID) { [void]$targetPids.Add([int]$item.ListenPID) }
            if ($null -ne $item.LauncherPID) { [void]$targetPids.Add([int]$item.LauncherPID) }
            if ($null -ne $item.PID) { [void]$targetPids.Add([int]$item.PID) }
        }
    } catch {
        Write-Host "[stop-frontend] warn: parse pid file failed: $pidFilePath"
    }
}

foreach ($p in $targetPids) {
    Stop-ByPid -TargetPid $p
}

foreach ($port in $Ports) {
    Stop-ByPort -Port $port
}

$alive = @()
foreach ($port in $Ports) {
    $listen = Get-NetTCPConnection -State Listen -LocalPort $port -ErrorAction SilentlyContinue | Select-Object -First 1
    if ($listen) {
        $alive += [PSCustomObject]@{ Port = $port; PID = $listen.OwningProcess }
    }
}

if ($alive.Count -eq 0) {
    Write-Host "[stop-frontend] all frontend ports stopped"
} else {
    Write-Host "[stop-frontend] some ports are still listening:"
    $alive | ForEach-Object { Write-Host ("  - :{0} pid={1}" -f $_.Port, $_.PID) }
}

