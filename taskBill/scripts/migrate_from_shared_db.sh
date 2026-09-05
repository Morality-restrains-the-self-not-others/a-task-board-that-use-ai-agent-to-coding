#!/usr/bin/env bash
# 一次性：从 Saas_project 共享库复制 billing 表到 taskBill/data/billing.db
# DEPRECATED (2026-08-24, docs-cleanup): 存储已全部 MySQL 化（db/registry.yaml driver: mysql），saas.sqlite3 共享库已随 Django v57 退役删除；本脚本为 SQLite 时代一次性迁移/清理工具，仅供历史参考，勿再执行。
set -euo pipefail
ROOT="$(cd "$(dirname "$0")/../.." && pwd)"
SHARED="${SHARED_DB:-$ROOT/db/saas/saas.sqlite3}"
BILL_DB="${BILL_DB:-$ROOT/db/task-bill/billing.sqlite3}"

mkdir -p "$(dirname "$BILL_DB")"
rm -f "$BILL_DB"

sqlite3 "$BILL_DB" < "$ROOT/taskBill/migrations/001_billing_tables.sql"
sqlite3 "$BILL_DB" "INSERT INTO taskbill_schema_migrations (name, applied_at) VALUES ('001_billing_tables.sql', datetime('now'));"

sqlite3 "$BILL_DB" <<SQL
ATTACH '$SHARED' AS src;
INSERT OR REPLACE INTO billing_pricing_package (
  id, package_number, valid_from, valid_to,
  normal_task_points, programming_task_points,
  normal_task_renewal_points_per_month, programming_task_renewal_points_per_month,
  created_at, updated_at
) SELECT
  id, package_number, valid_from, valid_to,
  normal_task_points, programming_task_points,
  normal_task_renewal_points_per_month, programming_task_renewal_points_per_month,
  created_at, updated_at
FROM src.billing_pricing_package;
INSERT OR REPLACE INTO billing_unit SELECT * FROM src.billing_unit;
INSERT OR REPLACE INTO billing_account (
  id, tenant_id, balance, pricing_package_id,
  locked_post_creation_points, locked_server_start_points,
  locked_normal_renewal_points_per_month, locked_programming_renewal_points_per_month,
  excluded_pricing_package_ids, created_at, updated_at
) SELECT
  id, tenant_id, balance, pricing_package_id,
  locked_post_creation_points, locked_server_start_points,
  locked_normal_renewal_points_per_month, locked_programming_renewal_points_per_month,
  excluded_pricing_package_ids, created_at, updated_at
FROM src.billing_account;
INSERT OR REPLACE INTO billing_transaction (
  id, account_id, transaction_type, amount, balance_before, balance_after,
  points_source_type, project_id, user_id, workspace_id, task_id,
  billing_unit_id, usage_amount, description, transaction_id, created_at
) SELECT
  id, account_id, transaction_type, amount, balance_before, balance_after,
  points_source_type, project_id, user_id, workspace_id, task_id,
  billing_unit_id, usage_amount, description, transaction_id, created_at
FROM src.billing_transaction;
INSERT OR REPLACE INTO billing_usage (
  id, account_id, billing_unit_id, amount, project_id, user_id, workspace_id, task_id,
  description, usage_time
) SELECT
  id, account_id, billing_unit_id, amount, project_id, user_id, workspace_id, task_id,
  description, usage_time
FROM src.billing_usage;
INSERT OR REPLACE INTO billing_idempotency_key SELECT * FROM src.billing_idempotency_key;
INSERT OR REPLACE INTO billing_outbox_message SELECT * FROM src.billing_outbox_message;
DETACH src;
SQL

echo "Migrated billing tables to $BILL_DB"
