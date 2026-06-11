# Fetch AgentBase runtime list + logs (diagnostics).
param(
    [string]$RuntimeName = "",
    [int]$Limit = 100
)

$ErrorActionPreference = "Stop"
$RepoRoot = Split-Path (Split-Path $PSScriptRoot -Parent) -Parent

function Import-DotEnv([string]$Path) {
    if (-not (Test-Path $Path)) { return }
    Get-Content $Path | ForEach-Object {
        $line = $_.Trim()
        if (-not $line -or $line.StartsWith("#")) { return }
        $eq = $line.IndexOf("=")
        if ($eq -lt 1) { return }
        $key = $line.Substring(0, $eq).Trim()
        $val = $line.Substring($eq + 1).Trim()
        if (-not [string]::IsNullOrWhiteSpace($key)) { Set-Item -Path "Env:$key" -Value $val }
    }
}

Import-DotEnv (Join-Path $RepoRoot ".env")
if (-not $env:GREENNODE_CLIENT_ID -or -not $env:GREENNODE_CLIENT_SECRET) {
    throw "Missing GREENNODE_CLIENT_ID / GREENNODE_CLIENT_SECRET in repo root .env"
}

$pair = "{0}:{1}" -f $env:GREENNODE_CLIENT_ID, $env:GREENNODE_CLIENT_SECRET
$basic = [Convert]::ToBase64String([Text.Encoding]::ASCII.GetBytes($pair))
$token = (Invoke-RestMethod -Method Post `
    -Uri "https://iam.api.vngcloud.vn/accounts-api/v2/auth/token" `
    -Headers @{ Authorization = "Basic $basic" } `
    -ContentType "application/x-www-form-urlencoded" `
    -Body "grant_type=client_credentials").access_token

$headers = @{ Authorization = "Bearer $token"; "Content-Type" = "application/json" }
$list = Invoke-RestMethod -Method Get -Uri "https://agentbase.api.vngcloud.vn/runtime/agent-runtimes?page=1&size=100" -Headers $headers

Write-Host "=== Runtimes ===" -ForegroundColor Cyan
foreach ($rt in $list.listData) {
    Write-Host ("  {0,-24} {1,-36} {2}" -f $rt.name, $rt.id, $rt.status)
}

$target = $null
if ($RuntimeName) {
    $target = $list.listData | Where-Object { $_.name -eq $RuntimeName } | Select-Object -First 1
} else {
    $target = $list.listData | Where-Object { $_.status -eq "ERROR" } | Select-Object -First 1
    if (-not $target) {
        $target = $list.listData | Where-Object { $_.name -like "opsone*" } | Select-Object -First 1
    }
}

if (-not $target) {
    Write-Host "No matching runtime found."
    exit 0
}

Write-Host ""
Write-Host "=== Logs: $($target.name) ($($target.id)) status=$($target.status) ===" -ForegroundColor Yellow
if ($target.statusReason) {
    Write-Host "statusReason: $($target.statusReason)" -ForegroundColor Yellow
}
$body = (@{ from = 0; limit = $Limit } | ConvertTo-Json -Compress)
$logs = Invoke-RestMethod -Method Post -Uri "https://agentbase.api.vngcloud.vn/runtime/agent-runtimes/$($target.id)/logs" -Headers $headers -Body $body -ContentType "application/json"
$entries = $logs.logs
if (-not $entries -and $logs.listData) { $entries = $logs.listData }
if ($entries) {
    foreach ($entry in $entries) {
        $ts = if ($entry.timestamp) { $entry.timestamp } else { "" }
        $msg = if ($entry.content) { $entry.content } elseif ($entry.message) { $entry.message } else { ($entry | ConvertTo-Json -Compress) }
        Write-Host "$ts $msg"
    }
} else {
    $logs | ConvertTo-Json -Depth 6
}
