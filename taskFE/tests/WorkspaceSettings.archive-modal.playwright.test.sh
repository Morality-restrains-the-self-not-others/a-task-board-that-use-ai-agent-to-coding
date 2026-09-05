#!/usr/bin/env bash
# Run WorkspaceSettings archive-modal E2E test (套餐设置打开任务存档模态)
# Usage: ./WorkspaceSettings.archive-modal.playwright.test.sh [--headed]
# Note: Uses bundled Chromium via executablePath override for environments without system Chrome.
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
PROJECT_DIR="$(cd "$SCRIPT_DIR/.." && pwd)"

cd "$PROJECT_DIR"

export CI="${CI:-1}"

npx playwright test \
  --config="$SCRIPT_DIR/WorkspaceSettings.archive-modal.playwright.config.js" \
  "$SCRIPT_DIR/WorkspaceSettings.archive-modal.playwright.test.js" \
  "$@"
