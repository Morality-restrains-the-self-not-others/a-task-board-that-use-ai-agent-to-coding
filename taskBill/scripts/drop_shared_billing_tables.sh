#!/usr/bin/env bash
# 从 Saas_project 共享库删除 billing_* 表（迁移至 taskBill 后执行，不做兼容）
# DEPRECATED (2026-08-24, docs-cleanup): 存储已全部 MySQL 化（db/registry.yaml driver: mysql），saas.sqlite3 共享库已随 Django v57 退役删除；本脚本为 SQLite 时代一次性迁移/清理工具，仅供历史参考，勿再执行。
set -euo pipefail
ROOT="$(cd "$(dirname "$0")/../.." && pwd)"
SHARED="${SHARED_DB:-$ROOT/db/saas/saas.sqlite3}"

sqlite3 "$SHARED" <<'SQL'
PRAGMA foreign_keys=OFF;
DROP TABLE IF EXISTS billing_outbox_message;
DROP TABLE IF EXISTS billing_idempotency_key;
DROP TABLE IF EXISTS billing_usage;
DROP TABLE IF EXISTS billing_transaction;
DROP TABLE IF EXISTS billing_account;
DROP TABLE IF EXISTS billing_unit;
DROP TABLE IF EXISTS billing_pricing_package;
DELETE FROM django_migrations WHERE app = 'billing';
PRAGMA foreign_keys=ON;
SQL

echo "Dropped billing_* tables from $SHARED"
