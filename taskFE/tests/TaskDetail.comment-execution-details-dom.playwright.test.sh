#!/usr/bin/env bash
set -euo pipefail
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
TASKFE_DIR="$(cd "${SCRIPT_DIR}/.." && pwd)"
cd "${TASKFE_DIR}"
exec npx playwright test "tests/TaskDetail.comment-execution-details-dom.playwright.test.js" --config="playwright.config.headless.js" "$@"
