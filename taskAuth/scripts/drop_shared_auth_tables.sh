#!/usr/bin/env bash
# Inc-5/Inc-6: 从 Saas_project 共享库删除所有 auth 表（身份认证已迁移至 taskAuth auth.sqlite3）
# DEPRECATED (2026-08-24, docs-cleanup): 存储已全部 MySQL 化（db/registry.yaml driver: mysql），saas.sqlite3 共享库已随 Django v57 退役删除；本脚本为 SQLite 时代一次性迁移/清理工具，仅供历史参考，勿再执行。
set -euo pipefail
ROOT="$(cd "$(dirname "$0")/../.." && pwd)"
SHARED="${SHARED_DB:-$ROOT/db/saas/saas.sqlite3}"

if [[ ! -f "$SHARED" ]]; then
  echo "Shared DB not found (MySQL migration — skip): $SHARED" >&2
  exit 0
fi

BACKUP="${SHARED}.pre-auth-drop.$(date +%Y%m%d%H%M%S).bak"
cp "$SHARED" "$BACKUP"
echo "Backup: $BACKUP"

sqlite3 "$SHARED" <<'SQL'
PRAGMA foreign_keys=OFF;
DROP TABLE IF EXISTS accounts_user;
DROP TABLE IF EXISTS accounts_customtoken;
DROP TABLE IF EXISTS accounts_login_method;
PRAGMA foreign_keys=ON;
SQL

echo "Dropped legacy auth tables (accounts_user/login_method/customtoken) from $SHARED"
