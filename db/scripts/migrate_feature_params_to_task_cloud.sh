#!/usr/bin/env bash
# 幂等：从 saas.sqlite3 拷贝 feature-params 相关表到 task_cloud.db
# 注意：saas Django 表列类型/FK 与 Cloud TEXT schema 不同，必须显式列名 + CAST，禁止 SELECT *
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/../.." && pwd)"
SAAS_DB="${SAAS_DATABASE_PATH:-$ROOT/db/saas/saas.sqlite3}"
CLOUD_DB="${TASK_CLOUD_DATABASE_PATH:-$ROOT/data/task_cloud.db}"
SCHEMA="$ROOT/db/feature_params/schema.sql"

if [[ ! -f "$SAAS_DB" ]]; then
  echo "[feature-params] saas db missing: $SAAS_DB" >&2
  exit 1
fi
if [[ ! -f "$SCHEMA" ]]; then
  echo "[feature-params] schema missing: $SCHEMA" >&2
  exit 1
fi

mkdir -p "$(dirname "$CLOUD_DB")"
sqlite3 "$CLOUD_DB" < "$SCHEMA"

copy_table() {
  local table="$1"
  local dest_cols="$2"
  local select_expr="$3"
  local exists
  exists="$(sqlite3 "$SAAS_DB" "SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name='$table';")"
  if [[ "$exists" != "1" ]]; then
    echo "[feature-params] skip missing saas table: $table"
    return 0
  fi
  sqlite3 "$CLOUD_DB" \
    ".timeout 30000" \
    "ATTACH DATABASE '$SAAS_DB' AS saas;" \
    "INSERT OR IGNORE INTO main.$table ($dest_cols) SELECT $select_expr FROM saas.$table;" \
    "DETACH DATABASE saas;"
  local n
  n="$(sqlite3 "$CLOUD_DB" "SELECT COUNT(*) FROM $table;")"
  echo "[feature-params] $table rows=$n"
}

TENANT_COLS='id, company_id, providers, agent_model, agent_model_provider, agent_max_steps, summary_model, summary_model_provider, llm_budget_enabled, extra_env_vars, created_at, updated_at'
TENANT_SEL="CAST(id AS TEXT), CAST(company_id AS TEXT), providers, agent_model, agent_model_provider, agent_max_steps, summary_model, summary_model_provider, CASE WHEN llm_budget_enabled THEN 1 ELSE 0 END, extra_env_vars, created_at, updated_at"
copy_table projects_tenant_feature_params "$TENANT_COLS" "$TENANT_SEL"

WS_COLS='id, workspace_id, company_id, use_company_default, providers, agent_model, agent_model_provider, agent_max_steps, summary_model, summary_model_provider, llm_budget_enabled, extra_env_vars, created_at, updated_at'
WS_SEL="CAST(id AS TEXT), workspace_id, company_id, CASE WHEN use_company_default THEN 1 ELSE 0 END, providers, agent_model, agent_model_provider, agent_max_steps, summary_model, summary_model_provider, CASE WHEN llm_budget_enabled THEN 1 ELSE 0 END, extra_env_vars, created_at, updated_at"
copy_table projects_workspace_feature_params "$WS_COLS" "$WS_SEL"

PERS_COLS='id, user_id, company_id, name, providers, agent_model, agent_model_provider, agent_max_steps, summary_model, summary_model_provider, llm_budget_enabled, extra_env_vars, created_at, updated_at'
PERS_SEL="CAST(id AS TEXT), user_id, company_id, name, providers, agent_model, agent_model_provider, agent_max_steps, summary_model, summary_model_provider, CASE WHEN llm_budget_enabled THEN 1 ELSE 0 END, extra_env_vars, created_at, updated_at"
copy_table projects_personal_feature_params_config "$PERS_COLS" "$PERS_SEL"

SNAP_COLS='id, task_id, workspace_id, tenant_id, source, source_config_id, source_display_name, resolved_env, providers_summary, agent_model, agent_model_provider, agent_max_steps, summary_model, summary_model_provider, created_at'
SNAP_SEL="CAST(id AS TEXT), task_id, workspace_id, tenant_id, source, source_config_id, source_display_name, resolved_env, providers_summary, agent_model, agent_model_provider, agent_max_steps, summary_model, summary_model_provider, created_at"
copy_table projects_task_feature_params_snapshot "$SNAP_COLS" "$SNAP_SEL"

USAGE_COLS='id, task_id, workspace_id, tenant_id, provider_name, api_key_hash, key_type, used_at, expires_at'
USAGE_SEL="CAST(id AS TEXT), task_id, workspace_id, tenant_id, provider_name, api_key_hash, key_type, used_at, expires_at"
copy_table projects_task_api_key_usage "$USAGE_COLS" "$USAGE_SEL"

AUDIT_COLS='id, user_id, company_id, workspace_id, resource, access_context, view_mode, auth_method, http_method, path, status_code, client_ip, user_agent, referer, trace_id, created_at'
AUDIT_SEL="CAST(id AS TEXT), user_id, company_id, workspace_id, resource, access_context, view_mode, auth_method, http_method, path, status_code, client_ip, user_agent, referer, trace_id, created_at"
copy_table projects_feature_params_access_audit "$AUDIT_COLS" "$AUDIT_SEL"

echo "[feature-params] migrated saas → $CLOUD_DB"
