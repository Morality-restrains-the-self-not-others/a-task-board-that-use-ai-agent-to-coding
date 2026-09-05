#!/usr/bin/env bash
# 工作面板控制台回归：经 CDP 9222 连接 Chrome，默认租户 850256677331562496。
# 用法：
#   bash playwright/tests/WorkPanel.console-regression.playwright.test.sh
# 环境变量（可选）：
#   PLAYWRIGHT_SITE_ORIGIN=http://183.250.1.132:4000
#   PLAYWRIGHT_TENANT_ID=850256677331562496
#   PLAYWRIGHT_TEST_EMAIL / PLAYWRIGHT_TEST_PASSWORD
set -euo pipefail
PW_ROOT="$(cd "$(dirname "$0")/../.." && pwd)"
cd "$PW_ROOT"

CDP_URL="${PW_CDP_URL:-http://127.0.0.1:9222}"
CHROME_BIN="${PW_CHROME_BIN:-}"
if [[ -z "$CHROME_BIN" ]]; then
  CHROME_BIN="$(find "${HOME}/.cache/ms-playwright" -path '*/chrome-linux64/chrome' 2>/dev/null | sort -r | head -1 || true)"
fi

if ! curl -sf "${CDP_URL}/json/version" >/dev/null 2>&1; then
  if [[ -n "$CHROME_BIN" && -x "$CHROME_BIN" ]]; then
    echo "[console-regression] starting headless Chrome on 9222..."
    "$CHROME_BIN" \
      --remote-debugging-port=9222 \
      --user-data-dir="${TMPDIR:-/tmp}/pw-chrome-console-regression" \
      --no-first-run \
      --no-default-browser-check \
      --no-sandbox \
      --headless=new \
      about:blank &
    for _ in $(seq 1 20); do
      curl -sf "${CDP_URL}/json/version" >/dev/null 2>&1 && break
      sleep 0.5
    done
  fi
fi

if ! curl -sf "${CDP_URL}/json/version" >/dev/null 2>&1; then
  echo "[console-regression] CDP not available at ${CDP_URL}; start Chrome with --remote-debugging-port=9222" >&2
  exit 1
fi

export PW_WS_ENDPOINT="$(
  CDP_URL="${CDP_URL}" python3 - <<'PY'
import json, os, urllib.request
url = os.environ["CDP_URL"].rstrip("/") + "/json/version"
print(json.load(urllib.request.urlopen(url, timeout=5))["webSocketDebuggerUrl"])
PY
)"

export PLAYWRIGHT_SITE_ORIGIN="${PLAYWRIGHT_SITE_ORIGIN:-http://183.250.1.132:4000}"
export PLAYWRIGHT_TENANT_ID="${PLAYWRIGHT_TENANT_ID:-850256677331562496}"
export PLAYWRIGHT_TEST_EMAIL="${PLAYWRIGHT_TEST_EMAIL:-contact@daydaymoney.com}"
export PLAYWRIGHT_TEST_PASSWORD="${PLAYWRIGHT_TEST_PASSWORD:-}"

npx playwright test -c playwright.config.chromium.js \
  tests/WorkPanel.console-regression.playwright.test.js \
  --project=chromium "$@"
