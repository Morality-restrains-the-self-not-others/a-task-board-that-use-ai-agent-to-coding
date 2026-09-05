#!/usr/bin/env bash
# Run BillingOrderCreate gitlab-region-single E2E test (购买页 GitLab 区域下拉唯一)
# Usage: ./BillingOrderCreate.gitlab-region-single.playwright.test.sh [--headed]
# Note: Uses bundled Chromium via executablePath override for environments without system Chrome.
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
PROJECT_DIR="$(cd "$SCRIPT_DIR/.." && pwd)"

cd "$PROJECT_DIR"

export CI="${CI:-1}"

npx playwright test \
  --config="$SCRIPT_DIR/BillingOrderCreate.gitlab-region-single.playwright.config.js" \
  "$SCRIPT_DIR/BillingOrderCreate.gitlab-region-single.playwright.test.js" \
  "$@"
