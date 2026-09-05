#!/usr/bin/env bash
# TaskDetail GitLab OAuth 绑定 E2E
# 默认跑 Doorkeeper 预检（无需登录）；全栈集成需 PLAYWRIGHT_INTEGRATION=1
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
PW_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"

cd "$PW_ROOT"

export PLAYWRIGHT_SITE_ORIGIN="${PLAYWRIGHT_SITE_ORIGIN:-http://127.0.0.1:4000}"
export PLAYWRIGHT_GATEWAY_ORIGIN="${PLAYWRIGHT_GATEWAY_ORIGIN:-http://127.0.0.1:18081}"
export GITLAB_BASE="${GITLAB_BASE:-http://127.0.0.1:8012}"
export GITLAB_LOCAL_REDIRECT_URI="${GITLAB_LOCAL_REDIRECT_URI:-http://183.250.1.132:18081/api/accounts/gitlab-local/oauth/callback/}"

if [[ "${PLAYWRIGHT_INTEGRATION:-}" == "1" ]]; then
  echo "[integration] 运行 TaskDetail GitLab OAuth 绑定全栈 E2E"
  npx playwright test -c playwright.config.headless.js \
    "tests/TaskDetail.gitlab-oauth-bind.playwright.test.js" \
    --project=chromium-headless \
    "$@"
else
  echo "[preflight] 仅运行 GitLab Doorkeeper client_id 预检（设置 PLAYWRIGHT_INTEGRATION=1 跑全栈）"
  npx playwright test -c playwright.config.headless.js \
    "tests/TaskDetail.gitlab-oauth-bind.playwright.test.js" \
    --project=chromium-headless \
    --grep "Doorkeeper 预检" \
    "$@"
fi
