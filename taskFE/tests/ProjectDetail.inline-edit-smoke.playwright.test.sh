#!/usr/bin/env bash
# 项目详情页 — 标签/自动运行/镜像点击即编辑冒烟（CDP 9222 或 bundled Chromium）
# 用法：
#   bash playwright/tests/ProjectDetail.inline-edit-smoke.playwright.test.sh
set -euo pipefail
PW_ROOT="$(cd "$(dirname "$0")/../.." && pwd)"
cd "$PW_ROOT"

CDP_URL="${PW_CDP_URL:-http://127.0.0.1:9222}"
CHROME_BIN="${PW_CHROME_BIN:-}"
if [[ -z "$CHROME_BIN" ]]; then
  CHROME_BIN="$(find "${HOME}/.cache/ms-playwright" -path '*/chrome-linux64/chrome' 2>/dev/null | sort -r | head -1 || true)"
fi

USE_CDP=0
if [[ "${PW_FORCE_CDP:-}" == "1" ]] && curl -sf "${CDP_URL}/json/version" >/dev/null 2>&1; then
  USE_CDP=1
elif [[ "${PW_FORCE_CDP:-}" == "1" ]] && [[ -n "$CHROME_BIN" && -x "$CHROME_BIN" ]]; then
  echo "[inline-edit-smoke] starting headless Chrome on 9222..."
  "$CHROME_BIN" \
    --remote-debugging-port=9222 \
    --user-data-dir="${TMPDIR:-/tmp}/pw-chrome-inline-edit-smoke" \
    --no-first-run \
    --no-default-browser-check \
    --no-sandbox \
    --headless=new \
    about:blank &
  for _ in $(seq 1 20); do
    if curl -sf "${CDP_URL}/json/version" >/dev/null 2>&1; then
      USE_CDP=1
      break
    fi
    sleep 0.5
  done
fi

CONFIG="playwright.config.chromium.js"
if [[ "$USE_CDP" -eq 1 ]]; then
  export PW_WS_ENDPOINT="$(
    CDP_URL="${CDP_URL}" python3 - <<'PY'
import json, os, urllib.request
url = os.environ["CDP_URL"].rstrip("/") + "/json/version"
print(json.load(urllib.request.urlopen(url, timeout=5))["webSocketDebuggerUrl"])
PY
  )"
  CONFIG="playwright.config.cdp.js"
  echo "[inline-edit-smoke] using CDP ${CDP_URL}"
else
  echo "[inline-edit-smoke] CDP unavailable; falling back to bundled Chromium"
fi

if [[ -z "${PLAYWRIGHT_SITE_ORIGIN:-}" ]]; then
  if curl -sf "http://127.0.0.1:4000/" >/dev/null 2>&1; then
    export PLAYWRIGHT_SITE_ORIGIN="http://127.0.0.1:4000"
    echo "[inline-edit-smoke] local dev server detected; using ${PLAYWRIGHT_SITE_ORIGIN}"
  else
    export PLAYWRIGHT_SITE_ORIGIN="http://183.250.1.132:4000"
    echo "[inline-edit-smoke] using remote ${PLAYWRIGHT_SITE_ORIGIN}"
  fi
fi
export PLAYWRIGHT_TENANT_ID="${PLAYWRIGHT_TENANT_ID:-850256677331562496}"
export PLAYWRIGHT_TEST_EMAIL="${PLAYWRIGHT_TEST_EMAIL:-contact@daydaymoney.com}"
export PLAYWRIGHT_TEST_PASSWORD="${PLAYWRIGHT_TEST_PASSWORD:-}"

npx playwright test -c "$CONFIG" \
  tests/ProjectDetail.inline-edit-smoke.playwright.test.js \
  --project=chromium --timeout=120000 "$@"
