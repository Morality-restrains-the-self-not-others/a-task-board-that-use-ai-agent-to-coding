#!/usr/bin/env bash
# 运行 workspace_collaborators API E2E 测试
#
# 前提：
#   1. Django :8001 运行中
#   2. 外部 Chrome 已启动并监听 CDP :9222（或 CI 模式自动启动）
#   3. 设置环境变量 PLAYWRIGHT_TEST_EMAIL / PLAYWRIGHT_TEST_PASSWORD
#
# 用法：
#   cd taskFE
#   bash tests/WorkspaceAccess.collaborators-api-200.playwright.test.sh
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PLAYWRIGHT_DIR="$(cd "$SCRIPT_DIR/../.." && pwd)"

cd "$PLAYWRIGHT_DIR"

npx playwright test \
  -c playwright.config.js \
  tests/WorkspaceAccess.collaborators-api-200.playwright.test.js \
  "$@"
