# OpsOne E2E & DoD verification (Windows)
# Usage: .\scripts\e2e.ps1
# Full 3 agent cycles (~10 min): .\scripts\e2e.ps1 -Full
# Skip re-seed: .\scripts\e2e.ps1 -SkipSeed

param(
    [switch]$Full,
    [switch]$SkipSeed
)

$ErrorActionPreference = "Stop"
$ProjectRoot = Split-Path $PSScriptRoot -Parent
Set-Location $ProjectRoot

$devScript = Join-Path $PSScriptRoot "dev.ps1"
if (-not (Test-Path $devScript)) {
    throw "Missing dev.ps1: $devScript"
}
. $devScript

if (-not (Get-Command Invoke-OpsOneReset -ErrorAction SilentlyContinue)) {
    throw "dev.ps1 did not define Invoke-OpsOneReset - check scripts/dev.ps1 for parse errors"
}

if (-not $SkipSeed) {
    Write-Host "=== Seed MySQL (UTF-8) ===" -ForegroundColor Cyan
    Invoke-OpsOneReset
}

Write-Host "=== Unit tests ===" -ForegroundColor Cyan
go test ./internal/rules/... -count=1
if ($LASTEXITCODE -ne 0) { exit $LASTEXITCODE }

Write-Host "=== Integration + E2E (OPSONE_INTEGRATION=1) ===" -ForegroundColor Cyan
$env:OPSONE_INTEGRATION = "1"
if ($Full) {
    $env:OPSONE_E2E_FULL = "1"
    go test ./internal/e2e/... ./internal/api/... ./internal/tools/... -v -count=1 -timeout 20m
} else {
    go test ./internal/e2e/... ./internal/api/... -v -count=1
}
if ($LASTEXITCODE -ne 0) { exit $LASTEXITCODE }

Write-Host "=== DoD smoke ===" -ForegroundColor Cyan
docker exec opsone-mysql mysql --default-character-set=utf8mb4 -uapp -psecret opsone -e "SELECT COUNT(*) AS products FROM products;"
Write-Host "E2E Phase 7 OK" -ForegroundColor Green
