param(
    [int]$Steps = 1,
    [switch]$All
)

$ErrorActionPreference = "Stop"
$repoRoot = (Resolve-Path (Join-Path $PSScriptRoot "..\..\")).Path
$devEnvFile = Join-Path $repoRoot "configs\local\dev.env"

. (Join-Path $PSScriptRoot "common.ps1")
Load-DevEnv -Path $devEnvFile

function Resolve-Migrate {
    Write-Host "installing migrate with mysql tag..."
    & go install -tags "mysql" github.com/golang-migrate/migrate/v4/cmd/migrate@v4.18.3
    if ($LASTEXITCODE -ne 0) {
        throw "failed to install migrate"
    }
    $gopath = (& go env GOPATH)
    $candidate = Join-Path $gopath "bin\migrate.exe"
    if (Test-Path $candidate) {
        return $candidate
    }
    throw "migrate executable not found"
}

$migrate = Resolve-Migrate

$dbHost = if ($env:FLASHSALE_MYSQL_HOST) { $env:FLASHSALE_MYSQL_HOST } else { "localhost" }
$dbPort = if ($env:FLASHSALE_MYSQL_PORT) { [int]$env:FLASHSALE_MYSQL_PORT } else { 3306 }
$dbUser = if ($env:FLASHSALE_MYSQL_USER) { $env:FLASHSALE_MYSQL_USER } else { $env:FLASH_MYSQL_APP_USER }
$dbPass = if ($env:FLASHSALE_MYSQL_PASSWORD) { $env:FLASHSALE_MYSQL_PASSWORD } else { $env:FLASH_MYSQL_APP_PASSWORD }
$dbs = @("user", "admin", "product", "order", "seckill")

if ($Steps -lt 1) {
    throw "Steps must be >= 1"
}

Push-Location $repoRoot
try {
    foreach ($db in $dbs) {
        $name = "flash_$db"
        $path = "deploy/migrations/$db"
        $dsn = "mysql://${dbUser}:${dbPass}@tcp(${dbHost}:${dbPort})/${name}?multiStatements=true"
        if ($All) {
            Write-Host "migrate down all -> $name"
            & $migrate -path $path -database $dsn down
        } else {
            Write-Host "migrate down $Steps -> $name"
            & $migrate -path $path -database $dsn down $Steps
        }
        if ($LASTEXITCODE -ne 0) {
            throw "migrate down failed for $name with exit code $LASTEXITCODE"
        }
    }
} finally {
    Pop-Location
}

Write-Host "migrate-down finished"
