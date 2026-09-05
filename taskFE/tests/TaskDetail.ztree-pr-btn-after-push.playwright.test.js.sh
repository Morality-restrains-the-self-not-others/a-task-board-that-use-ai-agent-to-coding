#!/usr/bin/env bash
# zTree 推送并创建PR 后 PR 按钮 CDP 核验
set -euo pipefail
ROOT="$(cd "$(dirname "$0")/../.." && pwd)"
cd "$ROOT"
export CDP_URL="${PW_CDP_URL:-${CDP_URL:-http://127.0.0.1:9222}}"
export PLAYWRIGHT_BASE_URL="${PLAYWRIGHT_BASE_URL:-http://localhost:4000}"

if ! curl -sf "${CDP_URL}/json/version" >/dev/null 2>&1; then
  echo "ERROR: CDP 不可用: ${CDP_URL}" >&2
  exit 1
fi
if ! curl -sf "${PLAYWRIGHT_BASE_URL}/" >/dev/null 2>&1; then
  echo "ERROR: 站点不可用: ${PLAYWRIGHT_BASE_URL}" >&2
  exit 1
fi

echo "[ztree-pr-btn] CDP=${CDP_URL} BASE=${PLAYWRIGHT_BASE_URL}"
node tests/TaskDetail.ztree-pr-btn-after-push-cdp.mjs
