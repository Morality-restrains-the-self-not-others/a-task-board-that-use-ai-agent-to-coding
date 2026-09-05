#!/usr/bin/env bash
# 项目详情：关闭自动克隆时子仓 token_error 不阻断自动运行
set -euo pipefail
ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT"
npx playwright test --config=playwright.config.headless.js \
  tests/ProjectDetail.auto-run-git-gate-auto-clone-off.playwright.test.js "$@"
