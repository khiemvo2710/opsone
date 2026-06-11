# Deploy OpsOne API + workers + web dashboard to GreenNode AgentBase (4 runtimes).
param(
    [string]$Flavor = "runtime-s2-general-2x4",
    [string]$EnvFile = "",
    [string]$ApiEndpoint = "",
    [switch]$SkipBuild,
    [switch]$DryRun
)

$ErrorActionPreference = "Stop"
$scriptDir = $PSScriptRoot

foreach ($target in @("api", "worker-mock", "worker-agent", "web")) {
    Write-Host ""
    Write-Host "========== Deploy $target ==========" -ForegroundColor Cyan
    $invokeArgs = @{
        Target     = $target
        Flavor     = $Flavor
        EnvFile    = $EnvFile
        SkipBuild  = $SkipBuild
        DryRun     = $DryRun
    }
    if ($ApiEndpoint) { $invokeArgs.ApiEndpoint = $ApiEndpoint }
    & "$scriptDir\deploy-greennode.ps1" @invokeArgs
    if ($LASTEXITCODE -ne 0) { exit $LASTEXITCODE }
}

Write-Host ""
Write-Host "All 4 runtimes deployed (api, workers, web)." -ForegroundColor Green
