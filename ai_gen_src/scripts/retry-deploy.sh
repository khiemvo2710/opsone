#!/usr/bin/env bash
set -euo pipefail
export PATH="/c/D/Projects/opsone/tools:$PATH"
set -a
. /c/D/Projects/opsone/.env
set +a
cd /c/D/Projects/opsone/ai_gen_src
bash scripts/deploy-greennode.sh
