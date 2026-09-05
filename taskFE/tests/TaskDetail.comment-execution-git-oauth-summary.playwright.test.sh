#!/usr/bin/env bash
# Run TaskDetail comment-execution-git-oauth-summary E2E test (评论执行细节 Git OAuth 摘要 T17)
# Usage: ./TaskDetail.comment-execution-git-oauth-summary.playwright.test.sh [--headed]
# Note: Uses bundled Chromium via executablePath override for environments without system Chrome.
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
PROJECT_DIR="$(cd "$SCRIPT_DIR/.." && pwd)"

cd "$PROJECT_DIR"

export CI="${CI:-1}"

npx playwright test \
  --config="$SCRIPT_DIR/TaskDetail.comment-execution-git-oauth-summary.playwright.config.js" \
  "$SCRIPT_DIR/TaskDetail.comment-execution-git-oauth-summary.playwright.test.js" \
  "$@"
