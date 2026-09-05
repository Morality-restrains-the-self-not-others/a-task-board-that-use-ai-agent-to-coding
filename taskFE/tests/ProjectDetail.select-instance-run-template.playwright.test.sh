#!/usr/bin/env bash
# 项目详情页 — 从可用实例列表选节点保存运行模版
# 本地 Mock：bash playwright/tests/ProjectDetail.select-instance-run-template.playwright.test.sh
# 远程实站：SITE_BASE=http://183.250.1.132:4000 PLAYWRIGHT_GATEWAY_ORIGIN=http://183.250.1.132:18081 bash .../ProjectDetail.select-instance-run-template.playwright.test.sh
set -euo pipefail
PW_ROOT="$(cd "$(dirname "$0")/../.." && pwd)"
cd "$PW_ROOT"
npx playwright test -c playwright.config.chromium.js \
  tests/ProjectDetail.select-instance-run-template.playwright.test.js \
  --project=chromium "$@"
