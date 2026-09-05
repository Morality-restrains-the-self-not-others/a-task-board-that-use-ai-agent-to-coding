#!/usr/bin/env bash
# Run Projects GitLab sync E2E tests
# Usage:
#   ./Projects.gitlab-sync.playwright.test.sh
#   ./Projects.gitlab-sync.playwright.test.sh --headed
#   SITE_BASE=http://183.250.1.132:4000 ./Projects.gitlab-sync.playwright.test.sh
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
# 须在 taskFE/ 下跑，避免 monorepo 根与 taskFE 各装一份 @playwright/test 导致
# "test.describe() called in worker without Playwright Test package"
TASKFE_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"

cd "$TASKFE_ROOT"

export CI="${CI:-1}"

# 内网/直连 E2E 不走系统代理，避免 ERR_PROXY_CONNECTION_FAILED
export NO_PROXY="${NO_PROXY:-127.0.0.1,localhost,183.250.1.132}"
export no_proxy="${no_proxy:-$NO_PROXY}"
unset http_proxy https_proxy HTTP_PROXY HTTPS_PROXY ALL_PROXY all_proxy 2>/dev/null || true

if [ "${1:-}" = "--headed" ]; then
  export PW_HEADED=1
  export CI=""
  shift
  echo "Running in headed mode..."
fi

npx playwright test \
  --config="$SCRIPT_DIR/Projects.gitlab-sync.playwright.config.js" \
  "$SCRIPT_DIR/Projects.gitlab-sync.playwright.test.js" \
  "$@"
