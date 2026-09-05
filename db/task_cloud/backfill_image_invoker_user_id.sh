#!/usr/bin/env bash
# 回填 cloud_server_configs.image_invoker_user_id
# 从 comment_container_bindings + start events 推断首次启动者
# 用法:
#   bash db/task_cloud/backfill_image_invoker_user_id.sh [--dry-run]
# 环境变量:
#   TASK_CLOUD_DATABASE_PATH
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/../.." && pwd)"
CLOUD_DB="${TASK_CLOUD_DATABASE_PATH:-$ROOT/data/task_cloud.db}"
DRY_RUN=0

for arg in "$@"; do
  case "$arg" in
    --dry-run|-n) DRY_RUN=1 ;;
    -h|--help)
      echo "Usage: $0 [--dry-run]"
      echo "Backfill image_invoker_user_id in cloud_server_configs for idle-reuse"
      exit 0
      ;;
    *) echo "[backfill-invoker] unknown arg: $arg" >&2; exit 2 ;;
  esac
done

log() { echo "[backfill-invoker] $*"; }
require_cmd() { command -v "$1" >/dev/null 2>&1 || { echo "missing: $1" >&2; exit 1; }; }
require_cmd sqlite3

if [[ ! -f "$CLOUD_DB" ]]; then
  echo "[backfill-invoker] cloud db missing: $CLOUD_DB" >&2; exit 1
fi

SQL="
-- Rule: use company_member_id from first start event per task
UPDATE cloud_server_configs
SET image_invoker_user_id = (
  SELECT e.event_data ->> '$.company_member_id'
  FROM cloud_server_events e
  WHERE e.task_id = cloud_server_configs.task_id
    AND e.event_type = 'start'
    AND e.event_data ->> '$.company_member_id' IS NOT NULL
    AND e.event_data ->> '$.company_member_id' != ''
  ORDER BY e.created_at ASC
  LIMIT 1
)
WHERE (image_invoker_user_id = '' OR image_invoker_user_id IS NULL)
  AND EXISTS (
    SELECT 1 FROM cloud_server_events e2
    WHERE e2.task_id = cloud_server_configs.task_id
      AND e2.event_type = 'start'
      AND e2.event_data ->> '$.company_member_id' IS NOT NULL
      AND e2.event_data ->> '$.company_member_id' != ''
  );
"

if [[ "$DRY_RUN" == "1" ]]; then
  log "dry-run — checking backfill potential:"
  sqlite3 "$CLOUD_DB" "SELECT 'total empty invoker', COUNT(*) FROM cloud_server_configs WHERE image_invoker_user_id = '' OR image_invoker_user_id IS NULL;"
  sqlite3 "$CLOUD_DB" "SELECT 'has start event with member', COUNT(DISTINCT c.id) FROM cloud_server_configs c JOIN cloud_server_events e ON c.task_id = e.task_id WHERE e.event_type = 'start' AND json_extract(e.event_data, '\$.company_member_id') IS NOT NULL AND json_extract(e.event_data, '\$.company_member_id') != '' AND (c.image_invoker_user_id = '' OR c.image_invoker_user_id IS NULL);"
  sqlite3 "$CLOUD_DB" "SELECT 'non-empty invoker values', image_invoker_user_id, COUNT(*) FROM cloud_server_configs WHERE image_invoker_user_id != '' AND image_invoker_user_id IS NOT NULL GROUP BY image_invoker_user_id;"
  log "dry-run complete (no changes)"
else
  before=$(sqlite3 "$CLOUD_DB" "SELECT COUNT(*) FROM cloud_server_configs WHERE image_invoker_user_id = '' OR image_invoker_user_id IS NULL;")
  sqlite3 -bail "$CLOUD_DB" "$SQL"
  after=$(sqlite3 "$CLOUD_DB" "SELECT COUNT(*) FROM cloud_server_configs WHERE image_invoker_user_id = '' OR image_invoker_user_id IS NULL;")
  log "backfill complete: $before → $after empty (filled $((before - after)))"
  log "final distribution:"
  sqlite3 "$CLOUD_DB" "SELECT CASE WHEN image_invoker_user_id = '' OR image_invoker_user_id IS NULL THEN '(empty)' ELSE image_invoker_user_id END, COUNT(*) FROM cloud_server_configs GROUP BY 1 ORDER BY COUNT(*) DESC;"
fi
