#!/usr/bin/env bash
# 工作空间机器节点策略：设置模态 + work-panel 摘要（CDP 9222 或 bundled Chromium）
set -euo pipefail
PW_ROOT="$(cd "$(dirname "$0")/../.." && pwd)"
cd "$PW_ROOT"

CDP_URL="${PW_CDP_URL:-http://127.0.0.1:9222}"
USE_CDP=0
if curl -sf "${CDP_URL}/json/version" >/dev/null 2>&1; then
  USE_CDP=1
fi

CONFIG="playwright.chromium.config.js"
if [[ "$USE_CDP" -eq 1 ]]; then
  if [[ -f playwright.config.cdp.js ]]; then
    export PW_WS_ENDPOINT="$(
      CDP_URL="${CDP_URL}" python3 - <<'PY'
import json, os, urllib.request
url = os.environ["CDP_URL"].rstrip("/") + "/json/version"
print(json.load(urllib.request.urlopen(url, timeout=5))["webSocketDebuggerUrl"])
PY
    )"
    CONFIG="playwright.config.cdp.js"
    echo "[machine-policy-e2e] using CDP ${CDP_URL}"
  else
    echo "[machine-policy-e2e] CDP up but no playwright.config.cdp.js; using bundled Chromium"
  fi
else
  echo "[machine-policy-e2e] CDP unavailable; using bundled Chromium"
fi

if [[ -z "${PLAYWRIGHT_SITE_ORIGIN:-}" ]]; then
  if curl -sf "http://127.0.0.1:4000/" >/dev/null 2>&1; then
    export PLAYWRIGHT_SITE_ORIGIN="http://127.0.0.1:4000"
  else
    export PLAYWRIGHT_SITE_ORIGIN="https://www.daydaymoney.com"
  fi
fi
export PLAYWRIGHT_TENANT_ID="${PLAYWRIGHT_TENANT_ID:-850256677331562496}"
export PW_WORKSPACE_ID="${PW_WORKSPACE_ID:-${PLAYWRIGHT_WORKSPACE_ID:-}}"
export PLAYWRIGHT_TEST_EMAIL="${PLAYWRIGHT_TEST_EMAIL:-contact@daydaymoney.com}"
export PLAYWRIGHT_TEST_PASSWORD="${PLAYWRIGHT_TEST_PASSWORD:-}"

echo "[machine-policy-e2e] SITE=${PLAYWRIGHT_SITE_ORIGIN} TENANT=${PLAYWRIGHT_TENANT_ID}"

npx playwright test -c "$CONFIG" \
  tests/WorkspaceSettings.machine-policy-modal.playwright.test.js \
  tests/WorkPanel.workspace-machine-idle-summary.playwright.test.js \
  --project=chromium --timeout=120000 --workers=1 "$@"
