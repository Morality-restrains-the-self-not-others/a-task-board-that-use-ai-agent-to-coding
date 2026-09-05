#!/usr/bin/env bash
# 公网 www+api 跨子域 GitLab taskAuth SSO 回归（CDP Chrome）
# 用法：
#   bash playwright/tests/Gitlab.daydaymoney-cross-subdomain-sso.playwright.test.sh
set -euo pipefail
PW_ROOT="$(cd "$(dirname "$0")/../.." && pwd)"
cd "$PW_ROOT"

CDP_URL="${PW_CDP_URL:-http://127.0.0.1:9223}"
if ! curl -sf "${CDP_URL}/json/version" >/dev/null 2>&1; then
  CDP_URL="${PW_CDP_URL_FALLBACK:-http://127.0.0.1:9222}"
fi
if ! curl -sf "${CDP_URL}/json/version" >/dev/null 2>&1; then
  echo "[cross-subdomain-sso] CDP 不可用：请启动 Chrome --remote-debugging-port=9222" >&2
  exit 1
fi
echo "[cross-subdomain-sso] using CDP ${CDP_URL}"

export CDP_URL
export NO_PROXY="${NO_PROXY:-*}"
export http_proxy= https_proxy= all_proxy= HTTP_PROXY= HTTPS_PROXY= ALL_PROXY=
export E2E_SITE_ORIGIN="${E2E_SITE_ORIGIN:-https://www.daydaymoney.com}"
export GATEWAY_URL="${GATEWAY_URL:-https://api.daydaymoney.com}"
export GITLAB_URL="${GITLAB_URL:-https://gitlab.daydaymoney.com}"
export PLAYWRIGHT_BASE_URL="${PLAYWRIGHT_BASE_URL:-$E2E_SITE_ORIGIN}"
export PW_EMAIL="${PW_EMAIL:-${PLAYWRIGHT_TEST_EMAIL:-contact@daydaymoney.com}}"
export PW_PASSWORD="${PW_PASSWORD:-}"

node "tests/Gitlab.daydaymoney-cross-subdomain-sso-cdp.mjs"
