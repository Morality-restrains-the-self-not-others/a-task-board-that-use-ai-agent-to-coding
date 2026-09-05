#!/usr/bin/env bash
# 运行 ProjectDetail SSH Git 仓库分支预览 Playwright E2E（使用内置 Chromium）
set -euo pipefail
ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT"
npx playwright test --config=playwright.config.ssh-branch-preview.js "$@"
