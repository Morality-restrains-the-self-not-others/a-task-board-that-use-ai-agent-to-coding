#!/usr/bin/env bash
# E2E: CreateProject GitLab OAuth 完整授权流程测试
#
# 运行方式：
#   cd taskFE
#   bash tests/CreateProject.gitlab-oauth-full-flow.playwright.test.sh
#
# 可选环境变量：
#   SITE_BASE          主站地址 (默认 http://183.250.1.132:4000)
#   TEST_TENANT_ID     租户 ID (默认 850256677331562496)
#   TEST_REPO_URL      仓库地址 (默认 http://183.250.1.132:8012/example-user/valuestream.git)
#   GITLAB_BASE        GitLab 地址 (默认 http://183.250.1.132:8012)
#   PLAYWRIGHT_TEST_EMAIL     主站邮箱
#   PLAYWRIGHT_TEST_PASSWORD  主站密码
#   GITLAB_USERNAME           GitLab 用户名
#   GITLAB_PASSWORD           GitLab 密码
#   SMOKE_ONLY=1              仅运行 smoke test（不执行完整 OAuth）
#   DEBUG=1                   启用调试输出

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
PLAYWRIGHT_DIR="$(cd "$SCRIPT_DIR/../.." && pwd)"

cd "$PLAYWRIGHT_DIR"

SITE_BASE="${SITE_BASE:-http://183.250.1.132:4000}"
TEST_TENANT_ID="${TEST_TENANT_ID:-850256677331562496}"
TEST_REPO_URL="${TEST_REPO_URL:-http://183.250.1.132:8012/example-user/valuestream.git}"
GITLAB_BASE="${GITLAB_BASE:-http://183.250.1.132:8012}"
PLAYWRIGHT_TEST_EMAIL="${PLAYWRIGHT_TEST_EMAIL:-contact@daydaymoney.com}"
PLAYWRIGHT_TEST_PASSWORD="${PLAYWRIGHT_TEST_PASSWORD:-}"
GITLAB_USERNAME="${GITLAB_USERNAME:-$PLAYWRIGHT_TEST_EMAIL}"
GITLAB_PASSWORD="${GITLAB_PASSWORD:-$PLAYWRIGHT_TEST_PASSWORD}"
SMOKE_ONLY="${SMOKE_ONLY:-0}"
DEBUG="${DEBUG:-0}"

export SITE_BASE TEST_TENANT_ID TEST_REPO_URL GITLAB_BASE
export PLAYWRIGHT_TEST_EMAIL PLAYWRIGHT_TEST_PASSWORD
export GITLAB_USERNAME GITLAB_PASSWORD

echo "=========================================="
echo " GitLab OAuth E2E Test"
echo "=========================================="
echo " Site:     $SITE_BASE"
echo " Tenant:   $TEST_TENANT_ID"
echo " Repo:     $TEST_REPO_URL"
echo " GitLab:   $GITLAB_BASE"
echo " Email:    $PLAYWRIGHT_TEST_EMAIL"
echo " Smoke:    $SMOKE_ONLY"
echo " Debug:    $DEBUG"
echo "=========================================="

CONFIG_FILE="$SCRIPT_DIR/CreateProject.gitlab-oauth.config.js"
TEST_FILE="$SCRIPT_DIR/CreateProject.gitlab-oauth-full-flow.playwright.test.js"

if [ ! -f "$CONFIG_FILE" ]; then
    echo "ERROR: Config file not found: $CONFIG_FILE"
    exit 1
fi

if [ ! -f "$TEST_FILE" ]; then
    echo "ERROR: Test file not found: $TEST_FILE"
    exit 1
fi

EXTRA_ARGS=()
if [ "$SMOKE_ONLY" = "1" ]; then
    EXTRA_ARGS+=("--grep" "仅验证仓库校验")
fi

if [ "$DEBUG" = "1" ]; then
    EXTRA_ARGS+=("--debug")
fi

echo ""
echo "Running: npx playwright test -c $CONFIG_FILE $TEST_FILE --project=chromium-bundled ${EXTRA_ARGS[*]}"
echo ""

npx playwright test \
    -c "$CONFIG_FILE" \
    "$TEST_FILE" \
    --project=chromium-bundled \
    "${EXTRA_ARGS[@]}"

EXIT_CODE=$?

echo ""
echo "=========================================="
echo " Test completed with exit code: $EXIT_CODE"
echo " Results: $PLAYWRIGHT_DIR/tests/test_results/"
echo " Report:  $PLAYWRIGHT_DIR/tests/playwright-report-local/"
echo "=========================================="

exit $EXIT_CODE
