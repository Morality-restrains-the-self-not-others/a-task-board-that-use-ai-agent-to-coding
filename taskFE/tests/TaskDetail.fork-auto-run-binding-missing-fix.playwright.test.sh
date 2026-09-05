#!/usr/bin/env bash
# Fork 自动运行后不得再 BINDING_MISSING（CDP 9222 → www.daydaymoney.com）
set -euo pipefail
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
TASKFE_DIR="$(cd "${SCRIPT_DIR}/.." && pwd)"
cd "${TASKFE_DIR}"

if [[ "${PLAYWRIGHT_INTEGRATION:-}" != "1" ]]; then
  echo "[skip] 设置 PLAYWRIGHT_INTEGRATION=1 后跑公网 CDP 验收"
  exit 0
fi

CDP_URL="${CDP_URL:-http://127.0.0.1:9222}"
CHROME_BIN="${PW_CHROME_BIN:-}"
if [[ -z "$CHROME_BIN" ]]; then
  CHROME_BIN="$(find "${HOME}/.cache/ms-playwright" -path '*/chrome-linux64/chrome' 2>/dev/null | sort -r | head -1 || true)"
fi
if [[ -z "$CHROME_BIN" ]] && command -v google-chrome >/dev/null 2>&1; then
  CHROME_BIN="$(command -v google-chrome)"
fi

if ! curl -sf "${CDP_URL}/json/version" >/dev/null 2>&1; then
  if [[ -n "$CHROME_BIN" && -x "$CHROME_BIN" ]]; then
    echo "[fork-binding-missing] starting Chrome on 9222..."
    "$CHROME_BIN" \
      --remote-debugging-port=9222 \
      --user-data-dir="${TMPDIR:-/tmp}/pw-chrome-fork-binding-missing" \
      --no-first-run \
      --no-default-browser-check \
      --no-sandbox \
      --headless=new \
      about:blank &
    for _ in $(seq 1 30); do
      curl -sf "${CDP_URL}/json/version" >/dev/null 2>&1 && break
      sleep 0.5
    done
  fi
fi

if ! curl -sf "${CDP_URL}/json/version" >/dev/null 2>&1; then
  echo "[fork-binding-missing] CDP 9222 未就绪：请先启动 scripts/runDebugChrome.sh 或 Chrome --remote-debugging-port=9222" >&2
  exit 1
fi

export PW_WS_ENDPOINT="$(
  CDP_URL="${CDP_URL}" python3 - <<'PY'
import json, os, urllib.request
url = os.environ["CDP_URL"].rstrip("/") + "/json/version"
print(json.load(urllib.request.urlopen(url, timeout=5))["webSocketDebuggerUrl"])
PY
)"
export PLAYWRIGHT_SITE_ORIGIN="${PLAYWRIGHT_SITE_ORIGIN:-https://www.daydaymoney.com}"
export CDP_URL

exec npx playwright test "tests/TaskDetail.fork-auto-run-binding-missing-fix.playwright.test.js" \
  --config="playwright.config.cdp.js" --timeout=1920000 "$@"
