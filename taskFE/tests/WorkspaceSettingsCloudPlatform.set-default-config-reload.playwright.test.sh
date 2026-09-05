#!/usr/bin/env bash
# 运行：在 taskFE 根下执行：
#   bash tests/WorkspaceSettingsCloudPlatform.set-default-config-reload.playwright.test.sh
# 需设置 E2E_EMAIL、E2E_PASSWORD；可选 E2E_TENANT_ID
set -euo pipefail
PW_ROOT="$(cd "$(dirname "$0")/../.." && pwd)"
cd "$PW_ROOT"
npx playwright test -c playwright.config.js \
  tests/WorkspaceSettingsCloudPlatform.set-default-config-reload.playwright.test.js \
  --project=chromium "$@"
