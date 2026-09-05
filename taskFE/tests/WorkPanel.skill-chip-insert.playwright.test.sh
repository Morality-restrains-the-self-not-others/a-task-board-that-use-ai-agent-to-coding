#!/usr/bin/env bash
# Run WorkPanel skill-chip E2E test (创建任务技能 chip 点击写入完整 $镜像 /技能)
# Usage: ./WorkPanel.skill-chip-insert.playwright.test.sh [--headed]
# Note: Uses bundled Chromium via executablePath override for environments without system Chrome.
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
PROJECT_DIR="$(cd "$SCRIPT_DIR/.." && pwd)"

cd "$PROJECT_DIR"

export CI="${CI:-1}"

npx playwright test \
  --config="$SCRIPT_DIR/WorkPanel.skill-chip-insert.playwright.config.js" \
  "$SCRIPT_DIR/WorkPanel.skill-chip-insert.playwright.test.js" \
  "$@"
