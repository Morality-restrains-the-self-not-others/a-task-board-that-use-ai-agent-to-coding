#!/usr/bin/env bash
# Run TaskDetail comment-git-oauth-probe-once E2E test (评论区 AccessToken 一次探测 T18)
# Usage: ./TaskDetail.comment-git-oauth-probe-once.playwright.test.sh [--headed]
# Note: Uses bundled Chromium via executablePath override for environments without system Chrome.
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
PROJECT_DIR="$(cd "$SCRIPT_DIR/.." && pwd)"

cd "$PROJECT_DIR"

export CI="${CI:-1}"

npx playwright test \
  --config="$SCRIPT_DIR/TaskDetail.comment-git-oauth-probe-once.playwright.config.js" \
  "$SCRIPT_DIR/TaskDetail.comment-git-oauth-probe-once.playwright.test.js" \
  "$@"
