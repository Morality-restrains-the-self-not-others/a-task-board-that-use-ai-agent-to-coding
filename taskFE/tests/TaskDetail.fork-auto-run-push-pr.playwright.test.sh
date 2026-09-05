#!/usr/bin/env bash
# Fork 自动运行后应成功 push、创建 PR、出现评论回帖。
# 经 chromium.connectOverCDP(http://127.0.0.1:9222)；不要把 /json/version 的 ws 交给 connectOptions。
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
    echo "[fork-push-pr] starting Chrome on 9222..."
    "$CHROME_BIN" \
      --remote-debugging-port=9222 \
      --remote-allow-origins=* \
      --user-data-dir="${TMPDIR:-/tmp}/pw-chrome-fork-push-pr" \
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
  echo "[fork-push-pr] CDP 9222 未就绪：请先启动 Chrome --remote-debugging-port=9222" >&2
  exit 1
fi

export PLAYWRIGHT_SITE_ORIGIN="${PLAYWRIGHT_SITE_ORIGIN:-https://www.daydaymoney.com}"
export CDP_URL
# PLAYWRIGHT_WAIT_TASK_ID=task_xxx 时跳过 Fork，只等该任务 PR+回帖（避免误点已有 Fork 页）。

exec node tests/TaskDetail.fork-auto-run-push-pr.cdp.mjs "$@"
