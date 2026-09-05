#!/usr/bin/env bash
set -euo pipefail
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
TASKFE_DIR="$(cd "${SCRIPT_DIR}/.." && pwd)"
cd "${TASKFE_DIR}"

if ! curl -sf -m 2 http://127.0.0.1:9222/json/version >/dev/null; then
  echo "CDP 9222 未就绪：请先启动 scripts/runDebugChrome.sh 或 google-chrome --remote-debugging-port=9222" >&2
  exit 1
fi

export PLAYWRIGHT_SITE_ORIGIN="${PLAYWRIGHT_SITE_ORIGIN:-https://www.daydaymoney.com}"
export PW_CDP_URL="${PW_CDP_URL:-http://127.0.0.1:9222}"
export PW_RESET_EMAIL="${PW_RESET_EMAIL:-author@example.com}"
export PW_RESET_EVIDENCE_DIR="${PW_RESET_EVIDENCE_DIR:-/tmp/pw-reset-evidence}"
mkdir -p "${PW_RESET_EVIDENCE_DIR}"

# Test runner launches a second Chromium (connectOptions ≠ CDP). Drive 9222 directly.
node tests/AuthResetPassword.request-email.cdp.mjs
