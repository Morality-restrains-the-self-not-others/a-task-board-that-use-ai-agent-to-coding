#!/usr/bin/env bash
# 运行：在 taskFE 根下执行：
#   bash tests/WorkPanel.deliverable-filter-dropdown-task-no.playwright.test.sh
set -euo pipefail
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
TASKFE_DIR="$(cd "${SCRIPT_DIR}/.." && pwd)"
cd "${TASKFE_DIR}"
exec npx playwright test \
  tests/WorkPanel.deliverable-filter-dropdown-task-no.playwright.test.js \
  --config=playwright.config.headless.js \
  "$@"
