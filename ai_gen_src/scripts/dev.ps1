# OpsOne dev helpers (Windows PowerShell)
# Usage: . .\scripts\dev.ps1

$ProjectRoot = Split-Path $PSScriptRoot -Parent
Set-Location $ProjectRoot

Write-Host "Go: $(go version)"

function Test-OpsOneMysqlPing {
    docker compose exec -T mysql mysqladmin ping -h localhost -u root -prootsecret --silent *>$null
    return ($LASTEXITCODE -eq 0)
}

function Ensure-OpsOneMysqlDatabase {
    # Reused Docker volumes may lack opsone DB / app grants (entrypoint only runs on first init).
    $dbExists = docker exec opsone-mysql mysql -uroot -prootsecret -N -e `
        "SELECT COUNT(*) FROM information_schema.schemata WHERE schema_name='opsone';" 2>$null
    if ($LASTEXITCODE -ne 0 -or [string]::IsNullOrWhiteSpace($dbExists)) {
        throw "Cannot inspect MySQL schemas as root. Check: docker compose logs mysql"
    }
    if ([int]$dbExists.Trim() -eq 0) {
        Write-Host "Database opsone missing on existing volume - creating app user grants..."
        $bootstrapSql = 'CREATE DATABASE IF NOT EXISTS opsone CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci; CREATE USER IF NOT EXISTS ''app''@''%'' IDENTIFIED BY ''secret''; GRANT ALL PRIVILEGES ON opsone.* TO ''app''@''%''; FLUSH PRIVILEGES;'
        docker exec opsone-mysql mysql -uroot -prootsecret -e $bootstrapSql *>$null
        if ($LASTEXITCODE -ne 0) {
            throw "Failed to bootstrap database 'opsone'. Try: docker compose down -v"
        }
    }
}

function Wait-OpsOneMysqlReady {
    param([int]$TimeoutSec = 120)
    Write-Host "Waiting for MySQL (up to ${TimeoutSec}s)..."
    $deadline = (Get-Date).AddSeconds($TimeoutSec)
    while ((Get-Date) -lt $deadline) {
        if (Test-OpsOneMysqlPing) {
            Ensure-OpsOneMysqlDatabase
            docker exec opsone-mysql mysql --default-character-set=utf8mb4 -uapp -psecret opsone -e "SELECT 1" *>$null
            if ($LASTEXITCODE -eq 0) {
                Write-Host "MySQL is ready."
                return
            }
        }
        Start-Sleep -Seconds 2
    }
    throw "MySQL did not become ready within ${TimeoutSec}s. Try: docker compose down -v; then Invoke-OpsOneReset. Logs: docker compose logs mysql"
}

function Invoke-OpsOneMysql {
    param([string]$SqlFile)
    $localPath = Join-Path $ProjectRoot $SqlFile
    if (-not (Test-Path $localPath)) {
        throw "SQL file not found: $localPath"
    }
    $leaf = Split-Path $SqlFile -Leaf
    $containerPath = "/tmp/opsone-$leaf"
    # docker cp preserves UTF-8 bytes for seed.sql (Vietnamese labels)
    docker cp $localPath "opsone-mysql:${containerPath}"
    if ($LASTEXITCODE -ne 0) {
        throw "docker cp failed for $SqlFile"
    }
    docker exec opsone-mysql mysql --default-character-set=utf8mb4 -uapp -psecret opsone -e "source $containerPath"
    if ($LASTEXITCODE -ne 0) {
        throw "MySQL failed running $SqlFile (exit $LASTEXITCODE)"
    }
    docker exec opsone-mysql rm -f $containerPath 2>$null | Out-Null
}

function Invoke-OpsOneReset {
    docker compose up -d mysql
    if ($LASTEXITCODE -ne 0) {
        throw "docker compose up failed"
    }
    Wait-OpsOneMysqlReady
    Write-Host "Recreating schema (DROP + CREATE from db/schema.sql)..."
    Invoke-OpsOneMysql "db\schema.sql"
    Invoke-OpsOneMysql "db\seed.sql"
    docker exec opsone-mysql mysql --default-character-set=utf8mb4 -uapp -psecret opsone -e "SELECT product_code, label FROM products WHERE product_code IN ('ZING','GARENA');"
    if ($LASTEXITCODE -ne 0) {
        throw "Seed verification query failed"
    }
    Write-Host "DB reset complete. Restart worker-mock and worker-agent."
}

function Invoke-OpsOneClearRuntime {
    docker compose up -d mysql
    Wait-OpsOneMysqlReady
    Invoke-OpsOneMysql "db\clear_runtime.sql"
    Write-Host "Runtime cleared. Catalog, routing_config, agent_settings kept."
    Write-Host "Restart worker-mock and worker-agent to repopulate mock_metrics."
}

function Invoke-OpsOneTest {
    $env:OPSONE_INTEGRATION = "1"
    go test ./... -v
}

function Invoke-OpsOneE2E {
    param([switch]$Full, [switch]$SkipSeed)
    & (Join-Path $PSScriptRoot "e2e.ps1") @PSBoundParameters
}

function Start-OpsOneAPI {
    & (Join-Path $PSScriptRoot "run-api.ps1")
}

function Start-OpsOneMock {
    go run ./cmd/worker-mock
}

function Start-OpsOneAgent {
    go run ./cmd/worker-agent
}

function Start-OpsOneWeb {
    & (Join-Path $ProjectRoot "web\dev.ps1")
}

function Add-OpsOneNodeToPath {
    $nodeDir = "C:\Program Files\nodejs"
    if (Test-Path $nodeDir) {
        $env:Path = "$nodeDir;" + $env:Path
        Write-Host "Added Node.js to PATH for this session: $nodeDir"
    } else {
        Write-Host "Node.js not found - install: winget install OpenJS.NodeJS.LTS"
    }
}

function Open-OpsOneDevPort5173 {
    & (Join-Path $PSScriptRoot 'Open-OpsOneDevPort5173.ps1')
}

Write-Host "Loaded: Invoke-OpsOneReset (DROP+CREATE schema + seed), Invoke-OpsOneClearRuntime, Invoke-OpsOneTest, Invoke-OpsOneE2E, Start-OpsOneAPI, Start-OpsOneMock, Start-OpsOneAgent, Start-OpsOneWeb, Add-OpsOneNodeToPath, Open-OpsOneDevPort5173"

if ($MyInvocation.InvocationName -ne '.') {
    Write-Host ''
    Write-Host 'Luu y: ban vua chay .\dev.ps1 — cac ham se mat khi script ket thuc.' -ForegroundColor Yellow
    Write-Host 'Dung:  . .\scripts\dev.ps1   (co dau cham dau, tu thu muc ai_gen_src)' -ForegroundColor Yellow
    Write-Host 'Roi:   Invoke-OpsOneReset' -ForegroundColor Yellow
}
