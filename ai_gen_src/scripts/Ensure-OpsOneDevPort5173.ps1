# Cau hinh portproxy 5173 -> 3000 (Admin / UAC).
# Usage (Admin): . .\scripts\Ensure-OpsOneDevPort5173.ps1

#Requires -RunAsAdministrator

$ErrorActionPreference = 'Continue'
$root = Split-Path $PSScriptRoot -Parent
$marker = Join-Path $root '.vite-dev-port-proxy'
$log = Join-Path $env:TEMP 'opsone-port5173-fix.log'
Remove-Item $log -ErrorAction SilentlyContinue

function Write-LogLine {
    param([string]$Text)
    Add-Content -Path $log -Value $Text
    Write-Host $Text
}

function Invoke-NetshLine {
    param([string]$Line)
    Write-LogLine $Line
    cmd.exe /c "$Line 2>&1" | ForEach-Object { Write-LogLine $_ }
    return $LASTEXITCODE
}

function Ensure-IpHelper {
    $svc = Get-Service iphlpsvc -ErrorAction SilentlyContinue
    if ($svc -and $svc.Status -ne 'Running') {
        Write-LogLine 'Start ip helper (iphlpsvc)...'
        Start-Service iphlpsvc -ErrorAction SilentlyContinue
    }
}

function Set-Port5173Portproxy {
    param([int]$BackendPort = 3000)
    Ensure-IpHelper

    Invoke-NetshLine 'netsh interface portproxy delete v4tov4 listenaddress=127.0.0.1 listenport=5173' | Out-Null
    Invoke-NetshLine 'netsh interface portproxy delete v4tov4 listenaddress=0.0.0.0 listenport=5173' | Out-Null
    $addCode = Invoke-NetshLine "netsh interface portproxy add v4tov4 listenaddress=127.0.0.1 listenport=5173 connectaddress=127.0.0.1 connectport=$BackendPort"

    $show = cmd.exe /c 'netsh interface portproxy show v4tov4 2>&1' | Out-String
    Write-LogLine $show.Trim()
    if ($show -match '5173') {
        return $true
    }
    return ($addCode -eq 0)
}

Write-Host 'Cau hinh portproxy: localhost:5173 -> 127.0.0.1:3000' -ForegroundColor Cyan
if (-not (Set-Port5173Portproxy -BackendPort 3000)) {
    Write-Host "Portproxy THAT BAI. Log: $log" -ForegroundColor Red
    Read-Host 'Enter de dong'
    exit 1
}

Set-Content -Path $marker -Value '3000' -Encoding ascii
Write-Host ''
Write-Host 'OK. Truy cap http://localhost:5173 (Vite se chay tren :3000)' -ForegroundColor Green
Write-Host 'Chay: cd ai_gen_src; .\web\dev.ps1' -ForegroundColor Green
Read-Host 'Enter de dong'
exit 0
