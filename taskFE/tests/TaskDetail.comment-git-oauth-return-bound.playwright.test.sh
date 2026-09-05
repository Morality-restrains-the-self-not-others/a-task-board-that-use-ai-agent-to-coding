#!/usr/bin/env bash
# 任务详情 Git OAuth「去绑定」回流后芯片应为已绑定（CDP 9222）
set -euo pipefail
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
TASKFE_DIR="$(cd "${SCRIPT_DIR}/.." && pwd)"
cd "${TASKFE_DIR}"

if [[ "${PLAYWRIGHT_INTEGRATION:-}" != "1" ]]; then
  echo "[skip] 设置 PLAYWRIGHT_INTEGRATION=1 后跑公网 CDP 验收"
  exit 0
fi

exec npx playwright test "tests/TaskDetail.comment-git-oauth-return-bound.playwright.test.js" \
  --config="playwright.config.headless.js" "$@"
