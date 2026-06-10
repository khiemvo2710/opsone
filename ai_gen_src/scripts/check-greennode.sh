#!/usr/bin/env bash
# OpsOne — GreenNode platform health check (no secrets printed)
set -euo pipefail
export PATH="/c/D/Projects/opsone/tools:$PATH"
SKILLS="/c/D/Projects/opsone/greennode-agentbase-skills"
set -a
. /c/D/Projects/opsone/.env
set +a
cd "$SKILLS"

probe() {
  local name="$1"
  local method="$2"
  local url="$3"
  local token
  token=$(bash .claude/skills/agentbase/scripts/get_token.sh 2>/dev/null) || { echo "$name|AUTH_FAIL|-"; return; }
  local raw code body
  raw=$(curl -s -w $'\n%{http_code}' -X "$method" "$url" \
    -H "Authorization: Bearer $token" -H "Content-Type: application/json")
  code=$(echo "$raw" | tail -1)
  body=$(echo "$raw" | sed '$d')
  local detail
  detail=$(echo "$body" | jq -r 'if type=="object" then (.message // .error // .code // empty) else empty end' 2>/dev/null | head -1)
  [ -z "$detail" ] && detail=$(echo "$body" | tr '\n' ' ' | cut -c1-80)
  echo "$name|$code|${detail:-ok}"
}

echo "=== GreenNode diagnostics ==="
bash .claude/skills/agentbase/scripts/check_credentials.sh iam | sed 's/^/IAM: /'
bash .claude/skills/agentbase/scripts/check_credentials.sh llm | sed 's/^/LLM: /' || true

TOKEN=$(bash .claude/skills/agentbase/scripts/get_token.sh)
CID_PREFIX=$(echo "$GREENNODE_CLIENT_ID" | cut -c1-8)
echo "Client ID prefix: ${CID_PREFIX}..."

echo ""
echo "=== API probes ==="
probe "Runtime flavors" GET "https://agentbase.api.vngcloud.vn/runtime/flavors"
probe "Runtime list" GET "https://agentbase.api.vngcloud.vn/runtime/agent-runtimes?page=1&size=10"
probe "CR repository" GET "https://agentbase.api.vngcloud.vn/cr/api/v1/repository"
probe "CR credentials" GET "https://agentbase.api.vngcloud.vn/cr/api/v1/registry-credential"
probe "Identity agents" GET "https://agentbase.api.vngcloud.vn/identity/api/v1/agents?page=0&size=5"
probe "OpenClaw list" GET "https://agentbase.api.vngcloud.vn/runtime/openclaws?page=1&size=5"

echo ""
echo "=== Discovery summary ==="
bash .claude/skills/agentbase/scripts/discovery.sh all 2>&1 | sed -n '1,80p'
