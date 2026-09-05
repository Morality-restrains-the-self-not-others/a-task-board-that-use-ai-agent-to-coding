#!/usr/bin/env bash
# 幂等：从 saas.sqlite3 复制 cloud_cloudserverevent / cloud_accesskeyiamidassociation → task_cloud.db
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/../.." && pwd)"
SAAS_DB="${SAAS_DATABASE_PATH:-$ROOT/db/saas/saas.sqlite3}"
CLOUD_DB="${TASK_CLOUD_DATABASE_PATH:-$ROOT/data/task_cloud.db}"

if [[ ! -f "$SAAS_DB" ]]; then
  echo "[task-cloud] saas db missing: $SAAS_DB" >&2
  exit 1
fi

mkdir -p "$(dirname "$CLOUD_DB")"

sqlite3 "$CLOUD_DB" <<'SQL'
CREATE TABLE IF NOT EXISTS cloud_server_events (
  id TEXT PRIMARY KEY,
  company_id TEXT NOT NULL,
  workspace_id TEXT NOT NULL,
  task_id TEXT NOT NULL,
  company_member_id TEXT NOT NULL DEFAULT '',
  event_type TEXT NOT NULL,
  event_data TEXT NOT NULL DEFAULT '{}',
  status TEXT NOT NULL DEFAULT 'pending',
  error_message TEXT DEFAULT '',
  created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
  updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);
CREATE INDEX IF NOT EXISTS idx_cse_task ON cloud_server_events(task_id);
CREATE INDEX IF NOT EXISTS idx_cse_company ON cloud_server_events(company_id);
CREATE INDEX IF NOT EXISTS idx_cse_company_task_type ON cloud_server_events(company_id, task_id, event_type);
ALTER TABLE cloud_server_events ADD COLUMN comment_id TEXT NOT NULL DEFAULT '';
CREATE INDEX IF NOT EXISTS idx_cse_comment ON cloud_server_events(comment_id);
CREATE TABLE IF NOT EXISTS access_key_iam_associations (
  id TEXT PRIMARY KEY,
  cloud_platform_auth_id TEXT NOT NULL,
  access_key TEXT NOT NULL,
  iam_id TEXT NOT NULL,
  created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
  updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);
CREATE INDEX IF NOT EXISTS idx_akia_auth ON access_key_iam_associations(cloud_platform_auth_id);
SQL

copy_events() {
  local exists
  exists="$(sqlite3 "$SAAS_DB" "SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name='cloud_cloudserverevent';")"
  if [[ "$exists" != "1" ]]; then
    echo "[task-cloud] skip missing saas table: cloud_cloudserverevent"
    return 0
  fi
  sqlite3 "$CLOUD_DB" \
    ".timeout 30000" \
    "ATTACH DATABASE '$SAAS_DB' AS saas;" \
    "INSERT OR IGNORE INTO main.cloud_server_events
      (id, company_id, workspace_id, task_id, company_member_id, event_type, event_data, status, error_message, created_at, updated_at)
     SELECT
      CAST(id AS TEXT),
      CAST(company_id AS TEXT),
      workspace_id,
      task_id,
      company_member_id,
      event_type,
      event_data,
      status,
      COALESCE(error_message, ''),
      created_at,
      updated_at
     FROM saas.cloud_cloudserverevent;" \
    "DETACH DATABASE saas;"
  local n
  n="$(sqlite3 "$CLOUD_DB" "SELECT COUNT(*) FROM cloud_server_events;")"
  echo "[task-cloud] cloud_server_events rows=$n"
}

copy_iam() {
  local exists
  exists="$(sqlite3 "$SAAS_DB" "SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name='cloud_accesskeyiamidassociation';")"
  if [[ "$exists" != "1" ]]; then
    echo "[task-cloud] skip missing saas table: cloud_accesskeyiamidassociation"
    return 0
  fi
  sqlite3 "$CLOUD_DB" \
    ".timeout 30000" \
    "ATTACH DATABASE '$SAAS_DB' AS saas;" \
    "INSERT OR IGNORE INTO main.access_key_iam_associations
      (id, cloud_platform_auth_id, access_key, iam_id, created_at, updated_at)
     SELECT
      CAST(id AS TEXT),
      CAST(cloud_platform_auth_id_id AS TEXT),
      access_key,
      iam_id,
      created_at,
      updated_at
     FROM saas.cloud_accesskeyiamidassociation;" \
    "DETACH DATABASE saas;"
  local n
  n="$(sqlite3 "$CLOUD_DB" "SELECT COUNT(*) FROM access_key_iam_associations;")"
  echo "[task-cloud] access_key_iam_associations rows=$n"
}

copy_events
copy_iam

echo "[task-cloud] migrated saas → $CLOUD_DB"
