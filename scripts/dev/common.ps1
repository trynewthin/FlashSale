function Load-DevEnv {
    param(
        [Parameter(Mandatory = $true)]
        [string]$Path
    )

    if (!(Test-Path $Path)) {
        $examplePath = "$Path.example"
        if (Test-Path $examplePath) {
            throw "dev env file not found: $Path. Please copy $examplePath to $Path and fill values."
        }
        throw "dev env file not found: $Path"
    }

    foreach ($rawLine in Get-Content $Path) {
        $line = $rawLine.Trim()
        if ($line -eq "" -or $line.StartsWith("#")) {
            continue
        }

        $idx = $line.IndexOf("=")
        if ($idx -le 0) {
            continue
        }

        $name = $line.Substring(0, $idx).Trim()
        $value = $line.Substring($idx + 1).Trim()

        if (($value.StartsWith("'") -and $value.EndsWith("'")) -or ($value.StartsWith('"') -and $value.EndsWith('"'))) {
            $value = $value.Substring(1, $value.Length - 2)
        }

        Set-Item -Path ("Env:{0}" -f $name) -Value $value
    }

    if (-not $env:FLASHSALE_MYSQL_HOST) {
        $env:FLASHSALE_MYSQL_HOST = "localhost"
    }
    if (-not $env:FLASHSALE_MYSQL_PORT -and $env:FLASH_MYSQL_PORT) {
        $env:FLASHSALE_MYSQL_PORT = $env:FLASH_MYSQL_PORT
    }
    if (-not $env:FLASHSALE_REDIS_ADDR -and $env:FLASH_REDIS_PORT) {
        $env:FLASHSALE_REDIS_ADDR = "localhost:$($env:FLASH_REDIS_PORT)"
    }
    if (-not $env:FLASHSALE_KAFKA_BROKERS -and $env:FLASH_KAFKA_PORT) {
        $env:FLASHSALE_KAFKA_BROKERS = "localhost:$($env:FLASH_KAFKA_PORT)"
    }
}
