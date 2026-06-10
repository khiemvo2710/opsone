#!/usr/bin/env bash
set -euo pipefail
export PATH="/c/D/Projects/opsone/tools:$PATH"
cd /c/D/Projects/opsone/greennode-agentbase-skills
TOKEN=$(bash .claude/skills/agentbase/scripts/get_token.sh)
URL="https://agentbase.api.vngcloud.vn/cr/api/v1/repository"
echo "GET $URL"
curl -s -w "\nHTTP_CODE:%{http_code}\n" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  "$URL"
