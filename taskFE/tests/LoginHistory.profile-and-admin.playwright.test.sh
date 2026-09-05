#!/usr/bin/env bash
# 登录历史 Playwright：超管 mock href + 客户公网 IP 非 RFC1918
# Usage: ./LoginHistory.profile-and-admin.playwright.test.sh [--headed]
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
PROJECT_DIR="$(cd "$SCRIPT_DIR/.." && pwd)"
cd "$PROJECT_DIR"

export CI="${CI:-1}"
export NO_PROXY="${NO_PROXY:-*}"
export http_proxy= https_proxy= all_proxy= HTTP_PROXY= HTTPS_PROXY= ALL_PROXY=
export PLAYWRIGHT_TEST_EMAIL="${PLAYWRIGHT_TEST_EMAIL:-contact@daydaymoney.com}"
export PLAYWRIGHT_TEST_PASSWORD="${PLAYWRIGHT_TEST_PASSWORD:-}"
export PLAYWRIGHT_SITE_ORIGIN="${PLAYWRIGHT_SITE_ORIGIN:-https://www.daydaymoney.com}"

echo "[login-history-e2e] mock admin href + privacy gate"
npx playwright test \
  --config="$SCRIPT_DIR/LoginHistory.admin-href.playwright.config.js" \
  "$SCRIPT_DIR/LoginHistory.admin-href.playwright.test.js" \
  "$@"

if [[ -z "${PLAYWRIGHT_TEST_PASSWORD}" ]]; then
  echo "[login-history-e2e] skip live customer IP: PLAYWRIGHT_TEST_PASSWORD unset" >&2
  exit 1
fi

echo "[login-history-e2e] live customer IP on ${PLAYWRIGHT_SITE_ORIGIN}"
npx playwright test \
  --config="$SCRIPT_DIR/LoginHistory.customer-ip.playwright.config.js" \
  "$SCRIPT_DIR/LoginHistory.customer-ip.playwright.test.js" \
  "$@"
