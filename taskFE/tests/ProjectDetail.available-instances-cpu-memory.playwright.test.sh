#!/usr/bin/env bash
# 项目详情页 — 可用实例 CPU/内存展示回归
set -euo pipefail
PW_ROOT="$(cd "$(dirname "$0")/../.." && pwd)"
cd "$PW_ROOT"
npx playwright test -c playwright.config.chromium.js \
  tests/ProjectDetail.available-instances-cpu-memory.playwright.test.js \
  --project=chromium "$@"
