#!/usr/bin/env bash
# 公网 daydaymoney 登录：POST /api/auth/ 不得 403（deny-internal / djangoInternalApiBase）
# 用法：
#   bash playwright/tests/Login.daydaymoney-auth-api-not-403.playwright.test.sh
# 环境变量（可选）：
#   DAYDAYMONEY_LOGIN_URL / PLAYWRIGHT_TEST_EMAIL / PLAYWRIGHT_TEST_PASSWORD
#   PW_CDP_URL=http://127.0.0.1:9223（推荐无系统代理的 Chrome；9222 若走坏代理会失败）
set -euo pipefail
PW_ROOT="$(cd "$(dirname "$0")/../.." && pwd)"
cd "$PW_ROOT"

CDP_URL="${PW_CDP_URL:-http://127.0.0.1:9223}"
if ! curl -sf "${CDP_URL}/json/version" >/dev/null 2>&1; then
  CDP_URL="${PW_CDP_URL_FALLBACK:-http://127.0.0.1:9222}"
fi
if ! curl -sf "${CDP_URL}/json/version" >/dev/null 2>&1; then
  echo "[daydaymoney-login] CDP 不可用：请启动无代理 Chrome（--remote-debugging-port=9223）" >&2
  exit 1
fi
echo "[daydaymoney-login] using CDP ${CDP_URL}"

export CDP_URL
export NO_PROXY="${NO_PROXY:-*}"
export http_proxy= https_proxy= all_proxy= HTTP_PROXY= HTTPS_PROXY= ALL_PROXY=

node "tests/Login.daydaymoney-auth-api-not-403-cdp.mjs"
