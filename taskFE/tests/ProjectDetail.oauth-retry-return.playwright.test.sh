#!/usr/bin/env bash
# 项目详情「授权异常重试」回流本页（CDP 9222 优先，否则 bundled Chromium）
set -euo pipefail
ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT"

CDP_URL="${PW_CDP_URL:-http://127.0.0.1:9222}"
USE_CDP=0
if curl -sf "${CDP_URL}/json/version" >/dev/null 2>&1; then
  USE_CDP=1
fi

CONFIG="playwright.config.headless.js"
if [[ "$USE_CDP" -eq 1 ]]; then
  export PW_WS_ENDPOINT="$(
    CDP_URL="${CDP_URL}" python3 - <<'PY'
import json, os, urllib.request
url = os.environ["CDP_URL"].rstrip("/") + "/json/version"
print(json.load(urllib.request.urlopen(url, timeout=5))["webSocketDebuggerUrl"])
PY
  )"
  CONFIG="playwright.config.cdp.js"
  echo "[oauth-retry-return] using CDP ${CDP_URL}"
else
  echo "[oauth-retry-return] CDP unavailable; using bundled Chromium headless"
fi

if [[ -z "${PLAYWRIGHT_SITE_ORIGIN:-}" ]]; then
  if curl -sf "http://127.0.0.1:4000/" >/dev/null 2>&1; then
    export PLAYWRIGHT_SITE_ORIGIN="http://127.0.0.1:4000"
  elif curl -sf "https://www.daydaymoney.com/" >/dev/null 2>&1; then
    export PLAYWRIGHT_SITE_ORIGIN="https://www.daydaymoney.com"
  fi
  echo "[oauth-retry-return] site ${PLAYWRIGHT_SITE_ORIGIN:-unset}"
fi

export PLAYWRIGHT_TEST_EMAIL="${PLAYWRIGHT_TEST_EMAIL:-contact@daydaymoney.com}"
export PLAYWRIGHT_TEST_PASSWORD="${PLAYWRIGHT_TEST_PASSWORD:-}"

exec npx playwright test --config="$CONFIG" \
  tests/ProjectDetail.oauth-retry-return.playwright.test.js "$@"
