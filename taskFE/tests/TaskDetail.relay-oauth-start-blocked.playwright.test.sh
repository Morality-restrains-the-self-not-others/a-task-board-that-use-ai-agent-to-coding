#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
PW_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"

cd "$PW_ROOT"

# mock 套件（默认）：无需登录
npx playwright test -c playwright.config.headless.js \
  "tests/TaskDetail.relay-oauth-start-blocked.playwright.test.js" \
  --project=chromium-headless \
  --grep "mock" \
  "$@"
