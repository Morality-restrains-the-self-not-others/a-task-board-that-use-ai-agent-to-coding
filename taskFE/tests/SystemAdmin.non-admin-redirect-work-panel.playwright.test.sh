#!/usr/bin/env bash
# 非超管访问 /system-admin/ → 跳转 work-panel（公网 daydaymoney）
# 用法：
#   bash playwright/tests/SystemAdmin.non-admin-redirect-work-panel.playwright.test.sh
# 环境变量（可选）：
#   PLAYWRIGHT_SITE_ORIGIN=https://www.daydaymoney.com
#   E2E_NONADMIN_EMAIL / E2E_NONADMIN_PASSWORD
#   PW_CDP_URL=http://127.0.0.1:9223
set -euo pipefail
PW_ROOT="$(cd "$(dirname "$0")/../.." && pwd)"
cd "$PW_ROOT"

export NO_PROXY="${NO_PROXY:-*}"
export http_proxy= https_proxy= all_proxy= HTTP_PROXY= HTTPS_PROXY= ALL_PROXY=

export PLAYWRIGHT_SITE_ORIGIN="${PLAYWRIGHT_SITE_ORIGIN:-https://www.daydaymoney.com}"
export E2E_NONADMIN_EMAIL="${E2E_NONADMIN_EMAIL:-e2e.nonadmin.sysadmin.redirect@ljytest.com}"
export E2E_NONADMIN_PASSWORD="${E2E_NONADMIN_PASSWORD:-E2eNonAdmin!8677}"

CDP_URL="${PW_CDP_URL:-http://127.0.0.1:9223}"
if ! curl -sf "${CDP_URL}/json/version" >/dev/null 2>&1; then
  CDP_URL="${PW_CDP_URL_FALLBACK:-http://127.0.0.1:9222}"
fi
if ! curl -sf "${CDP_URL}/json/version" >/dev/null 2>&1; then
  echo "[system-admin-nonadmin] CDP 不可用：请启动 Chrome --remote-debugging-port=9223（或 9222）" >&2
  exit 1
fi
echo "[system-admin-nonadmin] using CDP ${CDP_URL}"
export CDP_URL

node tests/SystemAdmin.non-admin-redirect-work-panel-cdp.mjs
