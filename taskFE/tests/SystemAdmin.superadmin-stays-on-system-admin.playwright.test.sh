#!/usr/bin/env bash
# 超管访问 /system-admin/ 应停留（对照非超管跳转 work-panel）
# 用法：
#   bash playwright/tests/SystemAdmin.superadmin-stays-on-system-admin.playwright.test.sh
# 环境变量（可选）：
#   PLAYWRIGHT_SITE_ORIGIN=https://www.daydaymoney.com
#   E2E_SUPERADMIN_EMAIL / E2E_SUPERADMIN_PASSWORD
#   PW_CDP_URL=http://127.0.0.1:9223
set -euo pipefail
PW_ROOT="$(cd "$(dirname "$0")/../.." && pwd)"
cd "$PW_ROOT"

export NO_PROXY="${NO_PROXY:-*}"
export http_proxy= https_proxy= all_proxy= HTTP_PROXY= HTTPS_PROXY= ALL_PROXY=

export PLAYWRIGHT_SITE_ORIGIN="${PLAYWRIGHT_SITE_ORIGIN:-https://www.daydaymoney.com}"
export E2E_SUPERADMIN_EMAIL="${E2E_SUPERADMIN_EMAIL:-author@example.com}"
export E2E_SUPERADMIN_PASSWORD="${E2E_SUPERADMIN_PASSWORD:-}"

# 种子管理员密码已改为随机值（OPT-20260824-001，明文销毁），测试环境须显式重置
# 为已知测试密码才能登录。容器/密码可用 SEED_MYSQL_CONTAINER / SEED_MYSQL_PASSWORD 覆盖。
MYSQL_CONTAINER="${SEED_MYSQL_CONTAINER:-docker-mysql-mysql-1}"
MYSQL_PASSWORD="${SEED_MYSQL_PASSWORD:-root123456}"
if python3 tests/helpers/seed_e2e_account.py --superadmin "$MYSQL_CONTAINER" "$MYSQL_PASSWORD" 2>/dev/null; then
  echo "[system-admin-superadmin] 超管测试密码已就绪（E2E 环境）"
else
  echo "[system-admin-superadmin] ⚠ 超管密码重置失败（MySQL 不可达？）— 测试将尝试使用现有密码" >&2
fi

CDP_URL="${PW_CDP_URL:-http://127.0.0.1:9223}"
if ! curl -sf "${CDP_URL}/json/version" >/dev/null 2>&1; then
  CDP_URL="${PW_CDP_URL_FALLBACK:-http://127.0.0.1:9222}"
fi
if ! curl -sf "${CDP_URL}/json/version" >/dev/null 2>&1; then
  echo "[system-admin-superadmin] CDP 不可用：请启动 Chrome --remote-debugging-port=9223（或 9222）" >&2
  exit 1
fi
echo "[system-admin-superadmin] using CDP ${CDP_URL}"
export CDP_URL

node tests/SystemAdmin.superadmin-stays-on-system-admin-cdp.mjs
