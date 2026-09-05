#!/usr/bin/env bash
# 运行：在 taskFE 根下执行：
#   bash tests/WorkPanel.kanban-drag-updates-progress.playwright.test.sh
# 需本机 Vite + Django；可选环境变量见测试文件注释。
set -euo pipefail
PW_ROOT="$(cd "$(dirname "$0")/../.." && pwd)"
cd "$PW_ROOT"
npx playwright test -c playwright.config.js \
  tests/WorkPanel.kanban-drag-updates-progress.playwright.test.js \
  --project=chromium "$@"
