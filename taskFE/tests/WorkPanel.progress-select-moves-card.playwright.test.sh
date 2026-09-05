#!/usr/bin/env bash
# 运行：bash playwright/tests/WorkPanel.progress-select-moves-card.playwright.test.sh
set -euo pipefail
PW_ROOT="$(cd "$(dirname "$0")/../.." && pwd)"
cd "$PW_ROOT"
npx playwright test -c playwright.config.js \
  tests/WorkPanel.progress-select-moves-card.playwright.test.js \
  --project=chromium "$@"
