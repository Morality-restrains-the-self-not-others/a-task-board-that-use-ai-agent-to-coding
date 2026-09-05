#!/usr/bin/env bash
# 开票已开具/拒绝/专票上传 + 赠送页防误送 Playwright（OPT-20260823-055/056/065）
# Usage: ./Billing.invoice-and-grant.playwright.test.sh [--headed]
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
PROJECT_DIR="$(cd "$SCRIPT_DIR/.." && pwd)"
cd "$PROJECT_DIR"

export CI="${CI:-1}"
export NO_PROXY="${NO_PROXY:-*}"
export http_proxy= https_proxy= all_proxy= HTTP_PROXY= HTTPS_PROXY= ALL_PROXY=

npx playwright test \
  --config="$SCRIPT_DIR/Billing.invoice-and-grant.playwright.config.js" \
  "$@"
