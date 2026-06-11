# OpsOne API — load ai_gen_src/.env then run cmd/api
# Usage (from ai_gen_src): .\scripts\run-api.ps1

$ErrorActionPreference = "Stop"

$ProjectRoot = Split-Path $PSScriptRoot -Parent
Set-Location $ProjectRoot

function Import-OpsOneDotEnv {
    param([string]$Path = ".env")

    if (-not (Test-Path $Path)) {
        Write-Host "Missing $Path — copy from .env.example first:" -ForegroundColor Yellow
        Write-Host "  copy .env.example .env"
        exit 1
    }

    $loaded = 0
    foreach ($raw in Get-Content $Path) {
        $line = $raw.Trim()
        if ($line -eq "" -or $line.StartsWith("#")) {
            continue
        }
        $eq = $line.IndexOf("=")
        if ($eq -lt 1) {
            continue
        }
        $name = $line.Substring(0, $eq).Trim()
        $value = $line.Substring($eq + 1).Trim()
        if (
            ($value.StartsWith('"') -and $value.EndsWith('"')) -or
            ($value.StartsWith("'") -and $value.EndsWith("'"))
        ) {
            $value = $value.Substring(1, $value.Length - 2)
        }
        Set-Item -Path "Env:$name" -Value $value -Force
        $loaded++
    }

    Write-Host "Loaded $loaded env vars from $Path"
    if ($env:LLM_API_KEY) {
        $model = if ($env:LLM_MODEL) { $env:LLM_MODEL } else { "(default)" }
        Write-Host "LLM: enabled (model=$model)"
    } else {
        Write-Host "LLM: disabled — set LLM_API_KEY in .env for smart chat" -ForegroundColor Yellow
    }
}

function Get-OpsOneAPIPort {
    $addr = if ($env:API_ADDR) { $env:API_ADDR.Trim() } else { ":8080" }
    if ($addr -match ':(\d+)$') {
        return [int]$Matches[1]
    }
    return 8080
}

function Stop-OpsOnePortListener {
    param([int]$Port = 8080)

    $conns = Get-NetTCPConnection -LocalPort $Port -ErrorAction SilentlyContinue
    if (-not $conns) {
        Write-Host "Port $Port is free."
        return
    }

    $procIds = $conns | Select-Object -ExpandProperty OwningProcess -Unique | Where-Object { $_ -gt 0 }
    foreach ($procId in $procIds) {
        try {
            $proc = Get-Process -Id $procId -ErrorAction Stop
            Write-Host "Stopping PID $procId ($($proc.ProcessName)) on port $Port"
            Stop-Process -Id $procId -Force -ErrorAction Stop
        } catch {
            Write-Host "Could not stop PID $procId : $_" -ForegroundColor Yellow
        }
    }

    Start-Sleep -Seconds 1
    $remaining = Get-NetTCPConnection -LocalPort $Port -ErrorAction SilentlyContinue
    if ($remaining) {
        Write-Host "Warning: port $Port may still be in use." -ForegroundColor Yellow
    } else {
        Write-Host "Port $Port is free."
    }
}

Import-OpsOneDotEnv

$port = Get-OpsOneAPIPort
Stop-OpsOnePortListener -Port $port

$addr = if ($env:API_ADDR) { $env:API_ADDR } else { ":8080" }
Write-Host "Starting OpsOne API on $addr ..."
go run ./cmd/api
if ($LASTEXITCODE -ne 0) {
    exit $LASTEXITCODE
}
