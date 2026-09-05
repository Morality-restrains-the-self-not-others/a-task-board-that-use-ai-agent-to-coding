#!/usr/bin/env bash
# Run Onboarding company-creation E2E tests
# Usage: ./Onboarding.company-creation.playwright.test.sh [--headed]
#
# 前置：APISIX 路由 django-tenant-accounts 已部署（2026-07-29 fix）
# 若未部署，API 测试将因 404 失败。
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
PROJECT_DIR="$(cd "$SCRIPT_DIR/../.." && pwd)"

cd "$PROJECT_DIR"

if [ "${1:-}" = "--headed" ]; then
  export CI=""
  echo "Running in headed mode..."
else
  export CI="1"
  echo "Running in headless mode..."
fi

npx playwright test \
  --config="$SCRIPT_DIR/../playwright.config.local.js" \
  "$SCRIPT_DIR/Onboarding.company-creation.playwright.test.js" \
  "$@"
