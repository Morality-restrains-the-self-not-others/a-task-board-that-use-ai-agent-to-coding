#!/usr/bin/env bash
# 运行：在 taskFE 根下执行：
#   bash tests/WorkPanel.create-task-branch-datalist-no-reopen.playwright.test.sh
# 需本机 Vite + Django；可选环境变量 PLAYWRIGHT_TEST_EMAIL / PLAYWRIGHT_TEST_PASSWORD。
set -euo pipefail
PW_ROOT="$(cd "$(dirname "$0")/../.." && pwd)"
cd "$PW_ROOT"
npx playwright test -c playwright.config.js \
  tests/WorkPanel.create-task-branch-datalist-no-reopen.playwright.test.js \
  --project=chromium "$@"
