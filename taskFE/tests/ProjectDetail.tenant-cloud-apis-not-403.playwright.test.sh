#!/usr/bin/env bash
# 项目详情页：installed-images / cloud regions 不得 403（deny-internal / membership 误拦）
# 用法：
#   bash playwright/tests/ProjectDetail.tenant-cloud-apis-not-403.playwright.test.sh
set -euo pipefail
PW_ROOT="$(cd "$(dirname "$0")/../.." && pwd)"
cd "$PW_ROOT"

CDP_URL="${PW_CDP_URL:-http://127.0.0.1:9223}"
if ! curl -sf "${CDP_URL}/json/version" >/dev/null 2>&1; then
  CDP_URL="${PW_CDP_URL_FALLBACK:-http://127.0.0.1:9222}"
fi
if ! curl -sf "${CDP_URL}/json/version" >/dev/null 2>&1; then
  echo "[project-detail-cloud] CDP 不可用：请启动 Chrome --remote-debugging-port=9223" >&2
  exit 1
fi
echo "[project-detail-cloud] using CDP ${CDP_URL}"

export CDP_URL
export NO_PROXY="${NO_PROXY:-*}"
export http_proxy= https_proxy= all_proxy= HTTP_PROXY= HTTPS_PROXY= ALL_PROXY=

node "tests/ProjectDetail.tenant-cloud-apis-not-403-cdp.mjs"
