#!/usr/bin/env bash
# Run TaskDetail server-start-history empty-state E2E
# Usage: ./TaskDetail.server-start-history-empty-state.playwright.test.sh [--headed]
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
PROJECT_DIR="$(cd "$SCRIPT_DIR/.." && pwd)"

cd "$PROJECT_DIR"

export CI="${CI:-1}"

npx playwright test \
  --config="$SCRIPT_DIR/TaskDetail.server-start-history-empty-state.playwright.config.js" \
  "$SCRIPT_DIR/TaskDetail.server-start-history-empty-state.playwright.test.js" \
  "$@"
