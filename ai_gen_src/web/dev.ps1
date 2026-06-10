# OpsOne React dev - use when npm is not on PATH after installing Node.js
# Usage: cd web; .\dev.ps1

$ErrorActionPreference = "Stop"

function Find-OpsOneNodeDir {
    $candidates = @(
        "C:\Program Files\nodejs",
        "${env:ProgramFiles}\nodejs",
        "${env:ProgramFiles(x86)}\nodejs",
        "$env:LOCALAPPDATA\Programs\node"
    )
    foreach ($dir in $candidates) {
        if ($dir -and (Test-Path "$dir\npm.cmd")) {
            return $dir
        }
    }
    return $null
}

$nodeDir = Find-OpsOneNodeDir
if (-not $nodeDir) {
    Write-Host "Node.js not found (npm.cmd missing)."
    Write-Host "Install: winget install OpenJS.NodeJS.LTS"
    Write-Host "Then reopen the terminal or run: . ..\scripts\dev.ps1; Add-OpsOneNodeToPath"
    exit 1
}

# Refresh PATH for this session
$env:Path = "$nodeDir;" + [System.Environment]::GetEnvironmentVariable("Path", "Machine") + ";" + [System.Environment]::GetEnvironmentVariable("Path", "User")

Set-Location $PSScriptRoot

if (-not (Test-Path ".env")) {
    if (Test-Path ".env.example") {
        Copy-Item ".env.example" ".env"
        Write-Host "Created .env from .env.example"
    }
}

if (-not (Test-Path "node_modules")) {
    Write-Host "Running npm install (first time)..."
    & "$nodeDir\npm.cmd" install
    if ($LASTEXITCODE -ne 0) { exit $LASTEXITCODE }
}

Write-Host "OpsOne UI: http://localhost:5173 (API proxy to :8080)"
& "$nodeDir\npm.cmd" run dev
