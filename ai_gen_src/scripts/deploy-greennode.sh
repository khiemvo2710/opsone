#!/usr/bin/env bash
# Deploy OpsOne API to GreenNode AgentBase (PUBLIC). Run from ai_gen_src via Git Bash.
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
REPO="$(cd "$ROOT/.." && pwd)"
SKILLS="$REPO/.claude/skills/agentbase"
RUNTIME_NAME="${RUNTIME_NAME:-opsone-api}"
FLAVOR="${FLAVOR:-runtime-s2-general-2x4}"
ENV_FILE="${ENV_FILE:-$ROOT/.env.greennode}"
TAG="v$(date +%Y%m%d%H%M%S)"

cd "$REPO"
set -a
[ -f "$REPO/.env" ] && . "$REPO/.env"
set +a
bash "$SKILLS/scripts/check_credentials.sh" iam

REPO_JSON=$(bash "$SKILLS/scripts/cr.sh" repo get)
REGISTRY=$(echo "$REPO_JSON" | jq -r '.registryUrl // .data.registryUrl // empty')
REPO_NAME=$(echo "$REPO_JSON" | jq -r '.name // .data.name // empty')
if [ -z "$REGISTRY" ] || [ -z "$REPO_NAME" ]; then
  echo "Failed to read registry from CR API" >&2
  exit 1
fi

IMAGE="${REGISTRY}/${REPO_NAME}/${RUNTIME_NAME}:${TAG}"
echo "Image: $IMAGE"

cd "$ROOT"
docker build --platform linux/amd64 -t "$IMAGE" .

bash "$SKILLS/scripts/cr.sh" credentials docker-login
docker push "$IMAGE"

cd "$REPO"
EXISTING=$(bash "$SKILLS/scripts/runtime.sh" list | jq -r --arg n "$RUNTIME_NAME" '.listData[]? | select(.name==$n) | .id' | head -1)
ENV_ARGS=()
if [ -f "$ENV_FILE" ]; then
  ENV_ARGS=(--env-file "$ENV_FILE")
fi

if [ -n "$EXISTING" ]; then
  echo "Updating runtime $EXISTING"
  bash "$SKILLS/scripts/runtime.sh" update "$EXISTING" \
    --image "$IMAGE" \
    --flavor "$FLAVOR" \
    --from-cr \
    --network-mode PUBLIC \
    "${ENV_ARGS[@]}"
  RUNTIME_ID="$EXISTING"
else
  echo "Creating runtime $RUNTIME_NAME"
  OUT=$(bash "$SKILLS/scripts/runtime.sh" create \
    --name "$RUNTIME_NAME" \
    --image "$IMAGE" \
    --flavor "$FLAVOR" \
    --from-cr \
    --network-mode PUBLIC \
    --min-replicas 1 \
    --max-replicas 1 \
    --cpu-scale 50 \
    --mem-scale 50 \
    "${ENV_ARGS[@]}")
  echo "$OUT"
  RUNTIME_ID=$(echo "$OUT" | jq -r '.id // .data.id // empty')
fi

bash "$SKILLS/scripts/runtime.sh" get "$RUNTIME_ID"
ENDPOINT=$(bash "$SKILLS/scripts/runtime.sh" endpoints list "$RUNTIME_ID" | jq -r '.listData[]? | select(.name=="DEFAULT") | .url' | head -1)
echo ""
echo "Deployment complete"
echo "  Runtime:  $RUNTIME_NAME ($RUNTIME_ID)"
echo "  Image:    $IMAGE"
echo "  Endpoint: $ENDPOINT"
echo "  Health:   curl -s -o /dev/null -w '%{http_code}' \"${ENDPOINT}/health\""
