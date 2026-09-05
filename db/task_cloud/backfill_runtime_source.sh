#!/usr/bin/env bash
# 回填 cloud_server_config_histories.runtime_source 细分码
# 对 runtime_source = 'cloud_vm' 或空的历史行，按入口线索尽量回填
# 用法:
#   bash db/task_cloud/backfill_runtime_source.sh [--dry-run]
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
      echo "Backfill runtime_source in cloud_server_config_histories from cloud_vm → subtype"
      exit 0
      ;;
    *) echo "[backfill-runtime-source] unknown arg: $arg" >&2; exit 2 ;;
  esac
done

log() { echo "[backfill-runtime-source] $*"; }

require_cmd() { command -v "$1" >/dev/null 2>&1 || { echo "missing: $1" >&2; exit 1; }; }
require_cmd sqlite3

if [[ ! -f "$CLOUD_DB" ]]; then
  echo "[backfill-runtime-source] cloud db missing: $CLOUD_DB" >&2; exit 1
fi

SQL="
-- Rule 1: started_via='manual' → cloud_vm_manual
UPDATE cloud_server_config_histories
SET runtime_source = 'cloud_vm_manual'
WHERE (runtime_source = 'cloud_vm' OR runtime_source = '' OR runtime_source IS NULL)
  AND task_id IN (
    SELECT task_id FROM cloud_server_configs WHERE started_via = 'manual'
  );

-- Rule 2: comment_id links to comment_container_bindings → cloud_vm_comment_mention
UPDATE cloud_server_config_histories
SET runtime_source = 'cloud_vm_comment_mention'
WHERE (runtime_source = 'cloud_vm' OR runtime_source = '' OR runtime_source IS NULL)
  AND task_id IN (
    SELECT csc.task_id FROM cloud_server_configs csc
    JOIN comment_container_bindings ccb ON csc.task_id = ccb.task_id AND csc.comment_id = ccb.comment_id
    WHERE csc.comment_id != ''
  );

-- Rule 3: has launch_request_id AND start event exists → likely auto_run
-- (heuristic: if the task has start events and launch_request_id, it was likely queued by auto_run)
UPDATE cloud_server_config_histories
SET runtime_source = 'cloud_vm_auto_run'
WHERE (runtime_source = 'cloud_vm' OR runtime_source = '' OR runtime_source IS NULL)
  AND launch_request_id != ''
  AND task_id IN (
    SELECT DISTINCT task_id FROM cloud_server_events WHERE event_type = 'start'
  )
  AND runtime_source = 'cloud_vm';

-- Rule 4: remaining empty → default cloud_vm
UPDATE cloud_server_config_histories
SET runtime_source = 'cloud_vm'
WHERE runtime_source = '' OR runtime_source IS NULL;
"

if [[ "$DRY_RUN" == "1" ]]; then
  log "dry-run — showing counts that would change:"
  for label in "total pending" "→cloud_vm_manual (started_via)" "→cloud_vm_comment_mention (comment bind)" "→cloud_vm_auto_run (launch_req + events)"; do
    :
  done
  sqlite3 "$CLOUD_DB" "SELECT 'pending cloud_vm/empty', COUNT(*) FROM cloud_server_config_histories WHERE runtime_source = 'cloud_vm' OR runtime_source = '' OR runtime_source IS NULL;"
  sqlite3 "$CLOUD_DB" "SELECT '→ cloud_vm_manual', COUNT(*) FROM cloud_server_config_histories WHERE (runtime_source = 'cloud_vm' OR runtime_source = '' OR runtime_source IS NULL) AND task_id IN (SELECT task_id FROM cloud_server_configs WHERE started_via = 'manual');"
  sqlite3 "$CLOUD_DB" "SELECT '→ cloud_vm_comment_mention', COUNT(*) FROM cloud_server_config_histories WHERE (runtime_source = 'cloud_vm' OR runtime_source = '' OR runtime_source IS NULL) AND task_id IN (SELECT csc.task_id FROM cloud_server_configs csc JOIN comment_container_bindings ccb ON csc.task_id = ccb.task_id AND csc.comment_id = ccb.comment_id WHERE csc.comment_id != '');"
  sqlite3 "$CLOUD_DB" "SELECT '→ cloud_vm_auto_run', COUNT(*) FROM cloud_server_config_histories WHERE runtime_source = 'cloud_vm' AND launch_request_id != '' AND task_id IN (SELECT DISTINCT task_id FROM cloud_server_events WHERE event_type = 'start');"
  log "dry-run complete (no changes)"
else
  before=$(sqlite3 "$CLOUD_DB" "SELECT COUNT(*) FROM cloud_server_config_histories WHERE runtime_source = 'cloud_vm' OR runtime_source = '' OR runtime_source IS NULL;")
  sqlite3 -bail "$CLOUD_DB" "$SQL"
  after=$(sqlite3 "$CLOUD_DB" "SELECT COUNT(*) FROM cloud_server_config_histories WHERE runtime_source = 'cloud_vm' OR runtime_source = '' OR runtime_source IS NULL;")
  log "backfill complete: $before → $after pending (resolved $((before - after)))"
  log "final distribution:"
  sqlite3 "$CLOUD_DB" "SELECT runtime_source, COUNT(*) FROM cloud_server_config_histories GROUP BY runtime_source ORDER BY COUNT(*) DESC;"
fi
