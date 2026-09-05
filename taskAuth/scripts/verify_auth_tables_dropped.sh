#!/usr/bin/env bash
# Inc-5/Inc-6: 验证共享库中 auth 表已全部删除（身份认证已迁移至 taskAuth auth.sqlite3）
# DEPRECATED (2026-08-24, docs-cleanup): 存储已全部 MySQL 化（db/registry.yaml driver: mysql），saas.sqlite3 共享库已随 Django v57 退役删除；本脚本为 SQLite 时代一次性迁移/清理工具，仅供历史参考，勿再执行。
set -euo pipefail
ROOT="$(cd "$(dirname "$0")/../.." && pwd)"
SHARED="${SHARED_DB:-$ROOT/db/saas/saas.sqlite3}"

if [[ ! -f "$SHARED" ]]; then
  echo "Shared DB not found (MySQL migration — skip): $SHARED" >&2
  exit 0
fi

for table in accounts_user accounts_login_method accounts_customtoken; do
  if sqlite3 "$SHARED" "SELECT 1 FROM sqlite_master WHERE type='table' AND name='$table';" | grep -q 1; then
    echo "FAIL: table still exists: $table" >&2
    exit 1
  fi
done

echo "OK: shared DB has no legacy auth tables (accounts_user/login_method/customtoken)"
