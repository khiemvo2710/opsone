# Deploy OpsOne to GreenNode AgentBase Runtime (PUBLIC).
# Targets: api | worker-mock | worker-agent | web  (or deploy-greennode-all.ps1 for all 4)
# Prerequisites: repo root .env with GREENNODE_CLIENT_ID/SECRET; ai_gen_src/.env.greennode for api/workers.
param(
    [ValidateSet("api", "worker-mock", "worker-agent", "web")]
    [string]$Target = "api",
    [string]$RuntimeName = "",
    [string]$Flavor = "runtime-s2-general-2x4",
    [string]$EnvFile = "",
    [string]$ApiEndpoint = "",
    [switch]$SkipBuild,
    [switch]$DryRun
)

$TargetMeta = @{
    api = @{
        Dockerfile  = "Dockerfile"
        BuildDir    = ""
        RuntimeName = "opsone-api"
        Description = "OpsOne REST API (chat LLM + dashboard backend)"
    }
    "worker-mock" = @{
        Dockerfile  = "Dockerfile.worker-mock"
        BuildDir    = ""
        RuntimeName = "opsone-worker-mock"
        Description = "OpsOne mock metrics worker (1m tick)"
    }
    "worker-agent" = @{
        Dockerfile  = "Dockerfile.worker-agent"
        BuildDir    = ""
        RuntimeName = "opsone-worker-agent"
        Description = "OpsOne scheduler agent (analysis + incidents)"
    }
    web = @{
        Dockerfile  = "Dockerfile"
        BuildDir    = "web"
        RuntimeName = "opsone-web"
        Description = "OpsOne React dashboard (nginx static + SPA)"
    }
}
if (-not $RuntimeName) { $RuntimeName = $TargetMeta[$Target].RuntimeName }
$Dockerfile = $TargetMeta[$Target].Dockerfile
$BuildDir = $TargetMeta[$Target].BuildDir
$Description = $TargetMeta[$Target].Description

$ErrorActionPreference = "Stop"
$AiGenSrc = Split-Path $PSScriptRoot -Parent
$RepoRoot = Split-Path $AiGenSrc -Parent
if (-not $EnvFile) { $EnvFile = Join-Path $AiGenSrc ".env.greennode" }

function Import-DotEnv([string]$Path) {
    if (-not (Test-Path $Path)) { return }
    Get-Content $Path | ForEach-Object {
        $line = $_.Trim()
        if (-not $line -or $line.StartsWith("#")) { return }
        $eq = $line.IndexOf("=")
        if ($eq -lt 1) { return }
        $key = $line.Substring(0, $eq).Trim()
        $val = $line.Substring($eq + 1).Trim()
        if (($val.StartsWith('"') -and $val.EndsWith('"')) -or ($val.StartsWith("'") -and $val.EndsWith("'"))) {
            $val = $val.Substring(1, $val.Length - 2)
        }
        if (-not [string]::IsNullOrWhiteSpace($key)) { Set-Item -Path "Env:$key" -Value $val }
    }
}

function Read-EnvFileMap([string]$Path) {
    $map = @{}
    if (-not (Test-Path $Path)) { return $map }
    Get-Content $Path | ForEach-Object {
        $line = $_.Trim()
        if (-not $line -or $line.StartsWith("#")) { return }
        $eq = $line.IndexOf("=")
        if ($eq -lt 1) { return }
        $key = $line.Substring(0, $eq).Trim()
        $val = $line.Substring($eq + 1).Trim()
        if (($val.StartsWith('"') -and $val.EndsWith('"')) -or ($val.StartsWith("'") -and $val.EndsWith("'"))) {
            $val = $val.Substring(1, $val.Length - 2)
        }
        if ($key -in @("GREENNODE_CLIENT_ID", "GREENNODE_CLIENT_SECRET", "GREENNODE_AGENT_IDENTITY", "GREENNODE_ENDPOINT_URL")) { return }
        if (-not [string]::IsNullOrWhiteSpace($key)) { $map[$key] = $val }
    }
    return $map
}

function Get-GreenNodeToken {
    if (-not $env:GREENNODE_CLIENT_ID -or -not $env:GREENNODE_CLIENT_SECRET) {
        throw "Missing GREENNODE_CLIENT_ID / GREENNODE_CLIENT_SECRET. Copy .env.example to repo root .env and fill IAM service account credentials."
    }
    $pair = "{0}:{1}" -f $env:GREENNODE_CLIENT_ID, $env:GREENNODE_CLIENT_SECRET
    $bytes = [Text.Encoding]::ASCII.GetBytes($pair)
    $basic = [Convert]::ToBase64String($bytes)
    $resp = Invoke-RestMethod -Method Post `
        -Uri "https://iam.api.vngcloud.vn/accounts-api/v2/auth/token" `
        -Headers @{ Authorization = "Basic $basic" } `
        -ContentType "application/x-www-form-urlencoded" `
        -Body "grant_type=client_credentials"
    if (-not $resp.access_token) { throw "Failed to obtain IAM token." }
    return $resp.access_token
}

function Invoke-GreenNodeApi {
    param(
        [string]$Method,
        [string]$Uri,
        [string]$Token,
        [object]$Body = $null
    )
    $headers = @{
        Authorization = "Bearer $Token"
        "Content-Type" = "application/json"
    }
    if ($null -ne $Body) {
        return Invoke-RestMethod -Method $Method -Uri $Uri -Headers $headers -Body ($Body | ConvertTo-Json -Depth 10 -Compress)
    }
    return Invoke-RestMethod -Method $Method -Uri $Uri -Headers $headers
}

function Get-RuntimeEndpointUrl {
    param(
        [string]$Token,
        [string]$Name
    )
    $list = Invoke-GreenNodeApi -Method Get -Uri "https://agentbase.api.vngcloud.vn/runtime/agent-runtimes?page=1&size=100" -Token $Token
    $rt = $null
    if ($list.listData) {
        $rt = $list.listData | Where-Object { $_.name -eq $Name -and $_.status -eq "ACTIVE" } | Select-Object -First 1
    }
    if (-not $rt) { return "" }
    $eps = Invoke-GreenNodeApi -Method Get -Uri "https://agentbase.api.vngcloud.vn/runtime/agent-runtimes/$($rt.id)/endpoints?page=1&size=20" -Token $Token
    $ep = ($eps.listData | Where-Object { $_.name -eq "DEFAULT" } | Select-Object -First 1).url
    return [string]$ep
}

Write-Host "=== OpsOne $Target -> GreenNode AgentBase (PUBLIC) ===" -ForegroundColor Cyan
Import-DotEnv (Join-Path $RepoRoot ".env")
Import-DotEnv (Join-Path $RepoRoot ".greennode.json") # no-op if missing; json not parsed here

if (-not $env:GREENNODE_CLIENT_ID -and (Test-Path (Join-Path $RepoRoot ".greennode.json"))) {
    $gn = Get-Content (Join-Path $RepoRoot ".greennode.json") -Raw | ConvertFrom-Json
    if ($gn.client_id) { $env:GREENNODE_CLIENT_ID = [string]$gn.client_id }
    if ($gn.client_secret) { $env:GREENNODE_CLIENT_SECRET = [string]$gn.client_secret }
}

if ($Target -ne "web" -and -not (Test-Path $EnvFile)) {
    throw "Container env file not found: $EnvFile`nCopy ai_gen_src/.env.greennode.example -> .env.greennode and set MYSQL_DSN (MySQL reachable from public internet)."
}

$token = Get-GreenNodeToken
Write-Host "IAM token: OK"

$webApiBase = ""
if ($Target -eq "web") {
    $apiEp = $ApiEndpoint
    if (-not $apiEp) {
        $apiEp = Get-RuntimeEndpointUrl -Token $token -Name "opsone-api"
    }
    if (-not $apiEp) {
        throw "opsone-api DEFAULT endpoint not found. Deploy api first or pass -ApiEndpoint https://..."
    }
    $webApiBase = ($apiEp.TrimEnd("/") + "/api/v1")
    Write-Host "Web will call API at: $webApiBase"
}

$repo = Invoke-GreenNodeApi -Method Get -Uri "https://agentbase.api.vngcloud.vn/cr/api/v1/repository" -Token $token
$registry = [string]$repo.registryUrl
$repoName = [string]$repo.name
if (-not $registry -or -not $repoName) { throw "Could not read Container Registry info." }

$cred = Invoke-GreenNodeApi -Method Get -Uri "https://agentbase.api.vngcloud.vn/cr/api/v1/registry-credential" -Token $token
$regUser = [string]$cred.username
$regPass = [string]$cred.secret
if (-not $regUser -or -not $regPass) { throw "Could not read CR docker credentials." }

$tag = "v{0:yyyyMMddHHmmss}" -f (Get-Date)
$image = "{0}/{1}/{2}:{3}" -f $registry, $repoName, $RuntimeName, $tag
Write-Host "Image: $image"

$buildContext = if ($BuildDir) { Join-Path $AiGenSrc $BuildDir } else { $AiGenSrc }
$dockerfilePath = Join-Path $buildContext $Dockerfile
$dockerBuildArgs = @("build", "-f", $dockerfilePath, "--platform", "linux/amd64", "-t", $image)
if ($Target -eq "web") {
    $dockerBuildArgs += @(
        "--build-arg", "VITE_API_BASE_URL=$webApiBase",
        "--build-arg", "VITE_DEV_AUTH_BYPASS=true"
    )
}
$dockerBuildArgs += $buildContext

if (-not $SkipBuild) {
    try {
        if ($DryRun) {
            Write-Host "[dry-run] docker $($dockerBuildArgs -join ' ')"
        } else {
            docker @dockerBuildArgs
            if ($LASTEXITCODE -ne 0) { throw "docker build failed" }
        }
    } catch {
        throw
    }

    if ($DryRun) {
        Write-Host "[dry-run] docker login $registry"
        Write-Host "[dry-run] docker push $image"
    } else {
        $regPass | docker login $registry -u $regUser --password-stdin | Out-Null
        if ($LASTEXITCODE -ne 0) { throw "docker login failed" }
        docker push $image
        if ($LASTEXITCODE -ne 0) { throw "docker push failed" }
    }
}

$envMap = if ($Target -eq "web") { @{} } else { Read-EnvFileMap $EnvFile }
$body = @{
    name = $RuntimeName
    description = $Description
    imageUrl = $image
    flavorId = $Flavor
    command = @()
    args = @()
    environmentVariables = $envMap
    autoscaling = @{
        minReplicas = 1
        maxReplicas = 1
        cpuUtilization = 50
        memoryUtilization = 50
    }
    networkConfig = @{ mode = "PUBLIC" }
    imageAuth = @{
        enabled = $true
        username = $regUser
        password = $regPass
    }
}

$list = Invoke-GreenNodeApi -Method Get -Uri "https://agentbase.api.vngcloud.vn/runtime/agent-runtimes?page=1&size=100" -Token $token
$existing = $null
if ($list.listData) {
    $existing = $list.listData | Where-Object { $_.name -eq $RuntimeName } | Select-Object -First 1
}

if ($DryRun) {
    Write-Host "[dry-run] would $(if ($existing) { 'update' } else { 'create' }) runtime $RuntimeName"
    return
}

if ($existing) {
    Write-Host "Updating runtime $($existing.id) ..."
    $result = Invoke-GreenNodeApi -Method Patch -Uri "https://agentbase.api.vngcloud.vn/runtime/agent-runtimes/$($existing.id)" -Token $token -Body $body
    $runtimeId = [string]$existing.id
} else {
    Write-Host "Creating runtime $RuntimeName ..."
    $result = Invoke-GreenNodeApi -Method Post -Uri "https://agentbase.api.vngcloud.vn/runtime/agent-runtimes" -Token $token -Body $body
    $runtimeId = [string]$result.id
}

if (-not $runtimeId) { throw "Runtime ID missing in API response." }

Write-Host "Waiting for ACTIVE ..."
$deadline = (Get-Date).AddMinutes(8)
$status = ""
do {
    Start-Sleep -Seconds 10
    $rt = Invoke-GreenNodeApi -Method Get -Uri "https://agentbase.api.vngcloud.vn/runtime/agent-runtimes/$runtimeId" -Token $token
    $status = [string]$rt.status
    Write-Host "  status: $status"
    if ($status -eq "ERROR") {
        if ($rt.statusReason) {
            Write-Host "  statusReason: $($rt.statusReason)" -ForegroundColor Red
        }
        Write-Host "Runtime logs (last 50 lines):" -ForegroundColor Red
        try {
            $logResp = Invoke-GreenNodeApi -Method Post -Uri "https://agentbase.api.vngcloud.vn/runtime/agent-runtimes/$runtimeId/logs" -Token $token -Body @{ from = 0; limit = 50 }
            $entries = $logResp.logs
            if (-not $entries -and $logResp.listData) { $entries = $logResp.listData }
            if ($entries) {
                foreach ($entry in $entries) {
                    $ts = if ($entry.timestamp) { $entry.timestamp } else { "" }
                    $msg = if ($entry.content) { $entry.content } elseif ($entry.message) { $entry.message } else { ($entry | ConvertTo-Json -Compress) }
                    Write-Host "  $ts $msg"
                }
            } else {
                Write-Host "  (no log entries returned)"
            }
        } catch {
            Write-Host "  (could not fetch logs: $($_.Exception.Message))"
        }
        throw "Runtime entered ERROR state. See logs above or AgentBase console."
    }
} while ($status -ne "ACTIVE" -and (Get-Date) -lt $deadline)

if ($status -ne "ACTIVE") { throw "Timed out waiting for ACTIVE (last: $status)." }

$eps = Invoke-GreenNodeApi -Method Get -Uri "https://agentbase.api.vngcloud.vn/runtime/agent-runtimes/$runtimeId/endpoints?page=1&size=20" -Token $token
$endpoint = ($eps.listData | Where-Object { $_.name -eq "DEFAULT" } | Select-Object -First 1).url

Write-Host ""
Write-Host "Deployment complete!" -ForegroundColor Green
Write-Host "  Runtime:   $RuntimeName ($runtimeId)"
Write-Host "  Image:     $image"
Write-Host "  Endpoint:  $endpoint"
Write-Host "  Health:    curl `"$endpoint/health`""
if ($Target -eq "web") {
    Write-Host "  Dashboard: $endpoint" -ForegroundColor Green
}
Write-Host "  Console:   https://aiplatform.console.vngcloud.vn/agent-runtime?tab=runtime"
