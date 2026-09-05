#!/usr/bin/env bash
# 工作空间功能参数切换保留配置 E2E 测试
# 前置条件：服务已启动，已部署 backend workspace_config 修复
set -euo pipefail

cd "$(dirname "$0")/.."

npx playwright test \
  -c playwright.verify.chromium.config.js \
  tests/WorkspaceFeatureParams.switch-preserve-config.playwright.test.js \
  --project=chromium \
  --reporter=list \
  "$@"
