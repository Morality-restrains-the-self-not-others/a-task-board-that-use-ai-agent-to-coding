#!/usr/bin/env bash
# Run SystemAdminUsers edit-user-id E2E test (编辑用户弹窗展示只读用户ID)
# Usage: ./SystemAdminUsers.edit-user-id.playwright.test.sh [--headed]
# Note: Uses bundled Chromium via executablePath override for environments without system Chrome.
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
PROJECT_DIR="$(cd "$SCRIPT_DIR/.." && pwd)"

cd "$PROJECT_DIR"

export CI="${CI:-1}"

npx playwright test \
  --config="$SCRIPT_DIR/SystemAdminUsers.edit-user-id.playwright.config.js" \
  "$SCRIPT_DIR/SystemAdminUsers.edit-user-id.playwright.test.js" \
  "$@"
