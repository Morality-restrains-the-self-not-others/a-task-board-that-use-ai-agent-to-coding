#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
PW_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"

cd "$PW_ROOT"

export PLAYWRIGHT_SITE_ORIGIN="${PLAYWRIGHT_SITE_ORIGIN:-http://183.250.1.132:4000}"
export PLAYWRIGHT_GATEWAY_ORIGIN="${PLAYWRIGHT_GATEWAY_ORIGIN:-http://183.250.1.132:18081}"
export PLAYWRIGHT_WORKSPACE_ID="${PLAYWRIGHT_WORKSPACE_ID:-861623708318031872}"
export PLAYWRIGHT_RELAY_TASK_ID="${PLAYWRIGHT_RELAY_TASK_ID:-task_12675381068363715869}"

npx playwright test -c playwright.config.chromium.js \
  "tests/TaskDetail.relay-token-init-clone.playwright.test.js" \
  "$@"
