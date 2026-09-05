#!/usr/bin/env bash
# 幂等：从 saas.sqlite3 INSERT…SELECT 四张 Budget 表到 task_budget.db
# 注意：saas Django 表列序与 schema.sql 不同，必须显式列名，禁止 SELECT *
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/../.." && pwd)"
SAAS_DB="${SAAS_DATABASE_PATH:-$ROOT/db/saas/saas.sqlite3}"
BUDGET_DB="${TASK_BUDGET_DATABASE_PATH:-$ROOT/db/task_budget/task_budget.db}"
SCHEMA="$ROOT/db/task_budget/schema.sql"

if [[ ! -f "$SAAS_DB" ]]; then
  echo "[task-budget] saas db missing (MySQL migration — skip): $SAAS_DB" >&2
  exit 0
fi
if [[ ! -s "$SAAS_DB" ]]; then
  echo "[task-budget] saas db empty — nothing to migrate, skip" >&2
  exit 0
fi
if [[ ! -f "$SCHEMA" ]]; then
  echo "[task-budget] schema missing: $SCHEMA" >&2
  exit 1
fi

# Check if saas has any budget tables before attempting migration
saas_tables=$(sqlite3 "$SAAS_DB" "SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name IN ('projects_workspace_model_budget_default','projects_task_model_budget','projects_task_model_budget_usage','projects_task_model_budget_usage_idempotency','projects_tenant_budget_permission');" 2>/dev/null || echo "0")
if [[ "$saas_tables" == "0" ]]; then
  echo "[task-budget] saas db has no budget tables — nothing to migrate, skip" >&2
  exit 0
fi

mkdir -p "$(dirname "$BUDGET_DB")"
sqlite3 "$BUDGET_DB" < "$SCHEMA"

copy_table() {
  local table="$1"
  local cols="$2"
  local exists
  exists="$(sqlite3 "$SAAS_DB" "SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name='$table';")"
  if [[ "$exists" != "1" ]]; then
    echo "[task-budget] skip missing saas table: $table"
    return 0
  fi
  sqlite3 "$BUDGET_DB" \
    ".timeout 30000" \
    "ATTACH DATABASE '$SAAS_DB' AS saas;" \
    "INSERT OR IGNORE INTO main.$table ($cols) SELECT $cols FROM saas.$table;" \
    "DETACH DATABASE saas;"
  local n
  n="$(sqlite3 "$BUDGET_DB" "SELECT COUNT(*) FROM $table;")"
  echo "[task-budget] $table rows=$n"
}

WS_COLS='id, workspace_id, company_id, provider, base_url, model_name, input_price_per_1m, output_price_per_1m, budget_limit, created_at, updated_at'
TASK_COLS='id, todo_id, workspace_id, company_id, provider, base_url, model_name, budget_limit, budget_limit_source, created_at, updated_at'
USAGE_COLS='id, todo_id, workspace_id, company_id, provider, base_url, model_name, input_tokens, output_tokens, spent_amount, last_reported_at, created_at, updated_at'
IDEM_COLS='id, idempotency_key, usage_id, created_at'

copy_table projects_workspace_model_budget_default "$WS_COLS"
copy_table projects_task_model_budget "$TASK_COLS"
copy_table projects_task_model_budget_usage "$USAGE_COLS"
copy_table projects_task_model_budget_usage_idempotency "$IDEM_COLS"

# tenant budget permissions（若 saas 仍有旧表）
PERM_COLS='id, company_id, subject_type, subject_id, can_raise_task_budget, created_at, updated_at'
# ensure destination table exists (Cloud also creates it at runtime)
sqlite3 "$BUDGET_DB" <<'SQL'
CREATE TABLE IF NOT EXISTS projects_tenant_budget_permission (
  id TEXT PRIMARY KEY,
  company_id TEXT NOT NULL,
  subject_type TEXT NOT NULL,
  subject_id TEXT NOT NULL,
  can_raise_task_budget INTEGER NOT NULL DEFAULT 0,
  created_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
  UNIQUE(company_id, subject_type, subject_id)
);
SQL
copy_table projects_tenant_budget_permission "$PERM_COLS"

echo "[task-budget] migrated saas → $BUDGET_DB"
