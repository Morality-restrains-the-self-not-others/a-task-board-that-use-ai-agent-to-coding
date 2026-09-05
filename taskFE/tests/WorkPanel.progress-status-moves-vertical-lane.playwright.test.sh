#!/usr/bin/env bash
# 运行：bash playwright/tests/WorkPanel.progress-status-moves-vertical-lane.playwright.test.sh
# 优先 CDP 9222；否则 bundled Chromium。需本机 4000 或 PLAYWRIGHT_SITE_ORIGIN。
set -euo pipefail
PW_ROOT="$(cd "$(dirname "$0")/../.." && pwd)"
cd "$PW_ROOT"

CDP_URL="${PW_CDP_URL:-http://127.0.0.1:9222}"
USE_CDP=0
if curl -sf "${CDP_URL}/json/version" >/dev/null 2>&1; then
  USE_CDP=1
fi

CONFIG="playwright.config.chromium.js"
if [[ "$USE_CDP" -eq 1 ]] && [[ -f playwright.config.cdp.js ]]; then
  export PW_WS_ENDPOINT="$(
    CDP_URL="${CDP_URL}" python3 - <<'PY'
import json, os, urllib.request
url = os.environ["CDP_URL"].rstrip("/") + "/json/version"
print(json.load(urllib.request.urlopen(url, timeout=5))["webSocketDebuggerUrl"])
PY
  )"
  CONFIG="playwright.config.cdp.js"
  echo "[progress-status-lane] using CDP ${CDP_URL}"
else
  echo "[progress-status-lane] using bundled Chromium (${CONFIG})"
fi

if [[ -z "${PLAYWRIGHT_SITE_ORIGIN:-}" ]]; then
  if curl -sf "http://127.0.0.1:4000/" >/dev/null 2>&1; then
    export PLAYWRIGHT_SITE_ORIGIN="http://127.0.0.1:4000"
  fi
fi

# 与目标页一致：有交付物×进度看板数据的租户
export PLAYWRIGHT_TENANT_ID="${PLAYWRIGHT_TENANT_ID:-850256677331562496}"
export PW_TENANT_ID="${PW_TENANT_ID:-850256677331562496}"
unset PW_WORKSPACE_ID PLAYWRIGHT_WORKSPACE_ID || true

npx playwright test -c "${CONFIG}" \
  tests/WorkPanel.progress-status-moves-vertical-lane.playwright.test.js \
  --project=chromium "$@"
