#!/usr/bin/env bash
# Playwright E2E: synology-gitlab OAuth 绑定流程
#
# 运行：
#   PLAYWRIGHT_INTEGRATION=1 bash playwright/tests/TaskDetail.synology-gitlab-oauth-bind.playwright.test.sh
#   PLAYWRIGHT_INTEGRATION=1 bash playwright/tests/TaskDetail.synology-gitlab-oauth-bind.playwright.test.sh --headed
#
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/../../../" && pwd)"

cd "$PROJECT_ROOT/playwright"

TEST_FILE="tests/TaskDetail.synology-gitlab-oauth-bind.playwright.test.js"
CONFIG_FILE="playwright.config.js"

HEADED_FLAG=""
if [[ "${1:-}" == "--headed" ]]; then
  HEADED_FLAG="--headed"
fi

echo "========================================="
echo "  synology-gitlab OAuth 绑定 E2E 测试"
echo "========================================="
echo ""

npx playwright test \
  -c "$CONFIG_FILE" \
  "$TEST_FILE" \
  --timeout=240000 \
  $HEADED_FLAG \
  "$@"
