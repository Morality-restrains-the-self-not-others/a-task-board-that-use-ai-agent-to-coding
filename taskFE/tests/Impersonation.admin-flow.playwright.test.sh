#!/usr/bin/env bash
# 系统管理模拟登录 Playwright：理由弹窗 / 收信箱 / 同目标重放 / 嵌套 409 中文
# Usage: ./Impersonation.admin-flow.playwright.test.sh [--headed]
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
PROJECT_DIR="$(cd "$SCRIPT_DIR/.." && pwd)"
cd "$PROJECT_DIR"

export CI="${CI:-1}"
export NO_PROXY="${NO_PROXY:-*}"
export http_proxy= https_proxy= all_proxy= HTTP_PROXY= HTTPS_PROXY= ALL_PROXY=

npx playwright test \
  --config="$SCRIPT_DIR/Impersonation.admin-flow.playwright.config.js" \
  "$SCRIPT_DIR/Impersonation.admin-flow.playwright.test.js" \
  "$@"
