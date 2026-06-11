#!/usr/bin/env bash
# Deploy OpsOne to GreenNode AgentBase (PUBLIC). TARGET=api|worker-mock|worker-agent|web
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
REPO="$(cd "$ROOT/.." && pwd)"
SKILLS="$REPO/.claude/skills/agentbase"
TARGET="${TARGET:-api}"
FLAVOR="${FLAVOR:-runtime-s2-general-2x4}"
ENV_FILE="${ENV_FILE:-$ROOT/.env.greennode}"
API_ENDPOINT="${API_ENDPOINT:-}"
TAG="v$(date +%Y%m%d%H%M%S)"
BUILD_DIR="$ROOT"
BUILD_ARGS=()

case "$TARGET" in
  api)
    DOCKERFILE="Dockerfile"
    RUNTIME_NAME="${RUNTIME_NAME:-opsone-api}"
    ;;
  worker-mock)
    DOCKERFILE="Dockerfile.worker-mock"
    RUNTIME_NAME="${RUNTIME_NAME:-opsone-worker-mock}"
    ;;
  worker-agent)
    DOCKERFILE="Dockerfile.worker-agent"
    RUNTIME_NAME="${RUNTIME_NAME:-opsone-worker-agent}"
    ;;
  web)
    DOCKERFILE="Dockerfile"
    BUILD_DIR="$ROOT/web"
    RUNTIME_NAME="${RUNTIME_NAME:-opsone-web}"
    ;;
  *)
    echo "TARGET must be api, worker-mock, worker-agent, or web" >&2
    exit 1
    ;;
esac

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

if [ "$TARGET" = "web" ]; then
  if [ -z "$API_ENDPOINT" ]; then
    API_RT=$(bash "$SKILLS/scripts/runtime.sh" list | jq -r '.listData[]? | select(.name=="opsone-api" and .status=="ACTIVE") | .id' | head -1)
    if [ -z "$API_RT" ]; then
      echo "opsone-api not ACTIVE. Deploy api first or set API_ENDPOINT." >&2
      exit 1
    fi
    API_ENDPOINT=$(bash "$SKILLS/scripts/runtime.sh" endpoints list "$API_RT" | jq -r '.listData[]? | select(.name=="DEFAULT") | .url' | head -1)
  fi
  API_BASE="${API_ENDPOINT%/}/api/v1"
  echo "Web will call API at: $API_BASE"
  BUILD_ARGS+=(--build-arg "VITE_API_BASE_URL=$API_BASE" --build-arg "VITE_DEV_AUTH_BYPASS=true")
fi

cd "$BUILD_DIR"
docker build -f "$DOCKERFILE" --platform linux/amd64 -t "$IMAGE" "${BUILD_ARGS[@]}" .

bash "$SKILLS/scripts/cr.sh" credentials docker-login
docker push "$IMAGE"

cd "$REPO"
EXISTING=$(bash "$SKILLS/scripts/runtime.sh" list | jq -r --arg n "$RUNTIME_NAME" '.listData[]? | select(.name==$n) | .id' | head -1)
ENV_ARGS=()
if [ "$TARGET" != "web" ] && [ -f "$ENV_FILE" ]; then
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
