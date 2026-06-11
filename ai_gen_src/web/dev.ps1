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

function Test-OpsOneDevPort {
    param([int]$Port = 5173)
    $listener = [System.Net.Sockets.TcpListener]::new([System.Net.IPAddress]::Loopback, $Port)
    try {
        $listener.Start()
        $listener.Stop()
        return $true
    } catch {
        return $false
    } finally {
        if ($listener.Server.IsBound) { $listener.Stop() }
    }
}

function Get-OpsOneDevPortBackend {
    $proxyMarker = Join-Path (Split-Path $PSScriptRoot -Parent) '.vite-dev-port-proxy'
    if (Test-Path $proxyMarker) {
        return (Get-Content $proxyMarker -Raw).Trim()
    }
    return $null
}

function Ensure-OpsOneDevPort {
    param([int]$Port = 5173)
    $backend = Get-OpsOneDevPortBackend
    if ($backend) {
        $env:VITE_DEV_PORT = $backend
        Write-Host "Portproxy: truy cap http://localhost:5173 (Vite tren :$backend)" -ForegroundColor Cyan
        return
    }

    if (Test-OpsOneDevPort -Port $Port) { return }

    $fixScript = Join-Path (Split-Path $PSScriptRoot -Parent) "scripts\Ensure-OpsOneDevPort5173.ps1"
    Write-Host ""
    Write-Host "Port $Port bi Windows/Hyper-V chan (EACCES)." -ForegroundColor Yellow
    Write-Host "Bam **Yes** tren cua UAC (mo port hoac portproxy 5173->3000)..."
    Start-Process -FilePath "powershell.exe" -Verb runAs -ArgumentList @(
        "-NoProfile", "-ExecutionPolicy", "Bypass", "-File", "`"$fixScript`""
    ) -Wait

    $backend = Get-OpsOneDevPortBackend
    if ($backend) {
        $env:VITE_DEV_PORT = $backend
        Write-Host "Portproxy: truy cap http://localhost:5173 (Vite tren :$backend)" -ForegroundColor Green
        return
    }

    if (Test-OpsOneDevPort -Port $Port) {
        Write-Host "Port $Port da san sang." -ForegroundColor Green
        return
    }

    Write-Host ""
    Write-Host "Chua mo duoc port $Port. Mo PowerShell (Admin), chay:" -ForegroundColor Yellow
    Write-Host "  . $fixScript" -ForegroundColor Cyan
    Write-Host ""
    exit 1
}

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

Ensure-OpsOneDevPort -Port 5173

$vitePort = if ($env:VITE_DEV_PORT) { $env:VITE_DEV_PORT } else { '5173' }
Write-Host "OpsOne UI: http://localhost:5173 (Vite :$vitePort, API proxy :8080)"
& "$nodeDir\npm.cmd" run dev
