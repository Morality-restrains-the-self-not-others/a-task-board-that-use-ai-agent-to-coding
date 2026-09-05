#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
PW_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"

cd "$PW_ROOT"

export PLAYWRIGHT_SITE_ORIGIN="${PLAYWRIGHT_SITE_ORIGIN:-http://127.0.0.1:4000}"
export PLAYWRIGHT_GATEWAY_ORIGIN="${PLAYWRIGHT_GATEWAY_ORIGIN:-http://127.0.0.1:18081}"
export PLAYWRIGHT_WORKSPACE_ID="${PLAYWRIGHT_WORKSPACE_ID:-861623708318031872}"
export PLAYWRIGHT_RELAY_TASK_ID="${PLAYWRIGHT_RELAY_TASK_ID:-task_12953905731855947865}"
export PLAYWRIGHT_TENANT_ID="${PLAYWRIGHT_TENANT_ID:-850256677331562496}"
export CDP_URL="${PW_CDP_URL:-http://127.0.0.1:9222}"

if [[ "${RUN_PW_TEST:-0}" == "1" ]]; then
  if curl -sf "${CDP_URL}/json/version" >/dev/null 2>&1; then
    export PW_WS_ENDPOINT="$(
      python3 -c "import json,urllib.request; print(json.load(urllib.request.urlopen('${CDP_URL}/json/version'))['webSocketDebuggerUrl'])"
    )"
    CONFIG="playwright.config.cdp.js"
  else
    CONFIG="playwright.config.chromium.js"
  fi
  exec npx playwright test -c "$CONFIG" \
    "tests/TaskDetail.relay-bootstrap-work-branch.playwright.test.js" \
    "$@"
fi

exec node tests/TaskDetail.relay-bootstrap-work-branch-cdp.mjs "$@"
