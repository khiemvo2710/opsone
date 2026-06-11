# Mo port 5173 qua portproxy - chay PowerShell thuong, BAM YES tren UAC.
# Usage: cd ai_gen_src; .\scripts\Open-OpsOneDevPort5173.ps1

$ErrorActionPreference = 'Stop'
$root = Split-Path $PSScriptRoot -Parent
$fixScript = Join-Path $PSScriptRoot 'Ensure-OpsOneDevPort5173.ps1'
$marker = Join-Path $root '.vite-dev-port-proxy'

if (Test-Path $marker) {
    Write-Host 'Portproxy da cau hinh. Chay: .\web\dev.ps1' -ForegroundColor Green
    exit 0
}

if (-not (Test-Path $fixScript)) {
    throw "Khong tim thay: $fixScript"
}

Write-Host ''
Write-Host '=== CAN BAM YES TREN CUA UAC ===' -ForegroundColor Yellow
Write-Host 'Windows se hoi quyen Admin (1 lan). Khong bam Yes = port 5173 khong dung duoc.' -ForegroundColor Yellow
Write-Host ''

$proc = Start-Process -FilePath 'powershell.exe' -Verb runAs -PassThru -ArgumentList @(
    '-NoProfile',
    '-ExecutionPolicy', 'Bypass',
    '-NoExit',
    '-File', "`"$fixScript`""
)

if (-not $proc) {
    Write-Host 'Khong mo duoc cua UAC. Thu PowerShell (Admin):' -ForegroundColor Red
    Write-Host "  . $fixScript" -ForegroundColor Cyan
    exit 1
}

$proc.WaitForExit()
$code = $proc.ExitCode

if (Test-Path $marker) {
    Write-Host 'Portproxy da cau hinh. Chay:' -ForegroundColor Green
    Write-Host '  .\web\dev.ps1' -ForegroundColor Cyan
    exit 0
}

Write-Host ''
Write-Host "Script Admin thoat ma loi (exit $code)." -ForegroundColor Red
Write-Host 'Mo PowerShell (Admin) va chay truc tiep:' -ForegroundColor Yellow
Write-Host "  cd $root" -ForegroundColor Cyan
Write-Host "  . .\scripts\Ensure-OpsOneDevPort5173.ps1" -ForegroundColor Cyan
Write-Host ''
Write-Host 'Log (neu co):' $env:TEMP\opsone-port5173-fix.log
exit 1
