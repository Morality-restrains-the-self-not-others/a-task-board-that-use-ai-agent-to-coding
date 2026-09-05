#!/usr/bin/env bash
# 执行日志滚动抖动回归（纯 mock，不依赖真实容器 / 网关）。
# 默认 bundled Chromium；若设置 USE_CDP=1 且 9222 可用则经 CDP 连接。
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
PW_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"
cd "$PW_ROOT"

TEST_FILE="tests/TaskDetail.exec-log-scroll-jitter.playwright.test.js"
export PLAYWRIGHT_SITE_ORIGIN="${PLAYWRIGHT_SITE_ORIGIN:-http://127.0.0.1:4000}"

CONFIG="playwright.config.chromium.js"
CDP_URL="${PW_CDP_URL:-http://127.0.0.1:9222}"

if [[ "${USE_CDP:-0}" == "1" ]] && curl -sf "${CDP_URL}/json/version" >/dev/null 2>&1; then
  export PW_WS_ENDPOINT="$(
    CDP_URL="${CDP_URL}" python3 - <<'PY'
import json, os, urllib.request
url = os.environ["CDP_URL"].rstrip("/") + "/json/version"
print(json.load(urllib.request.urlopen(url, timeout=5))["webSocketDebuggerUrl"])
PY
  )"
  CONFIG="playwright.config.cdp.js"
fi

echo "[exec-log-jitter] config=${CONFIG} site=${PLAYWRIGHT_SITE_ORIGIN}"
npx playwright test -c "${CONFIG}" "${TEST_FILE}" --project=chromium --workers=1 --timeout=120000 "$@"
