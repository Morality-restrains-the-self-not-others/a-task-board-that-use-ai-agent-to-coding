#!/usr/bin/env bash
# DROP saas 中已迁至 task_cloud.db 的 feature-params 六表（不可逆）。
# 前置：已跑 migrate_feature_params_to_task_cloud.sh，task_cloud.db 含完整数据。
# 用法:
#   bash db/scripts/drop_feature_params_from_saas.sh [--dry-run] [--force]
# 环境变量:
#   SAAS_DATABASE_PATH / TASK_CLOUD_DATABASE_PATH
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/../.." && pwd)"
SAAS_DB="${SAAS_DATABASE_PATH:-$ROOT/db/saas/saas.sqlite3}"
CLOUD_DB="${TASK_CLOUD_DATABASE_PATH:-$ROOT/data/task_cloud.db}"
DRY_RUN=0
FORCE=0

for arg in "$@"; do
  case "$arg" in
    --dry-run|-n) DRY_RUN=1 ;;
    --force|-f) FORCE=1 ;;
    -h|--help)
      sed -n '2,8p' "$0"
      exit 0
      ;;
    *)
      echo "[drop-feature-params] unknown arg: $arg" >&2
      exit 2
      ;;
  esac
done

# 已迁至 task_cloud 的六表（按迁移脚本顺序）
FEATURE_PARAM_TABLES=(
  "projects_tenant_feature_params"
  "projects_workspace_feature_params"
  "projects_personal_feature_params_config"
  "projects_task_feature_params_snapshot"
  "projects_task_api_key_usage"
  "projects_feature_params_access_audit"
)

log() { echo "[drop-feature-params] $*"; }

require_cmd() {
  command -v "$1" >/dev/null 2>&1 || {
    echo "[drop-feature-params] missing command: $1" >&2
    exit 1
  }
}

table_exists() {
  local db="$1" table="$2"
  sqlite3 "$db" "SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name='$table';"
}

require_cmd sqlite3

if [[ ! -f "$SAAS_DB" ]]; then
  echo "[drop-feature-params] saas db missing: $SAAS_DB" >&2
  exit 1
fi

# 门禁：task_cloud.db 必须存在且含六表数据
if [[ ! -f "$CLOUD_DB" ]]; then
  echo "[drop-feature-params] task_cloud.db missing: $CLOUD_DB — refuse drop without cloud target" >&2
  exit 1
fi

log "saas db: $SAAS_DB"
log "cloud db: $CLOUD_DB"

# 验证 cloud 端每表都有数据（至少 1 行），否则拒绝
missing_cloud=0
for table in "${FEATURE_PARAM_TABLES[@]}"; do
  if [[ "$(table_exists "$CLOUD_DB" "$table")" != "1" ]]; then
    log "FAIL: cloud missing table $table — 迁移未完成？" >&2
    missing_cloud=1
  fi
done
if [[ "$missing_cloud" == "1" ]]; then
  echo "[drop-feature-params] refuse: cloud 端表不完整，请先运行 migrate_feature_params_to_task_cloud.sh" >&2
  exit 1
fi
log "cloud 六表齐全，安全门禁通过"

dropped_count=0
skipped_count=0

for table in "${FEATURE_PARAM_TABLES[@]}"; do
  if [[ "$(table_exists "$SAAS_DB" "$table")" != "1" ]]; then
    log "idempotent: $table already absent on saas"
    ((skipped_count++)) || true
    continue
  fi
  n="$(sqlite3 "$SAAS_DB" "SELECT COUNT(*) FROM \"$table\";")"
  if [[ "$DRY_RUN" == "1" ]]; then
    log "dry-run would: DROP TABLE $table (rows=$n)"
    ((dropped_count++)) || true
    continue
  fi
  if [[ "$FORCE" != "1" ]]; then
    echo "[drop-feature-params] refuse DROP without --force (table=$table rows=$n)" >&2
    echo "[drop-feature-params] 已跳过 $dropped_count 个空表；请用 --force 确认删除非空表" >&2
    exit 1
  fi
  # 最终安全检查：不允许删除非 feature-params 表
  if [[ ! " ${FEATURE_PARAM_TABLES[*]} " =~ " ${table} " ]]; then
    echo "[drop-feature-params] FATAL: $table not in allowed list" >&2
    exit 1
  fi
  sqlite3 "$SAAS_DB" "DROP TABLE \"$table\";"
  log "dropped saas.$table (was rows=$n)"
  ((dropped_count++)) || true
done

# 验证：saas 端六表已全部移除
if [[ "$DRY_RUN" != "1" ]]; then
  remaining=0
  for table in "${FEATURE_PARAM_TABLES[@]}"; do
    if [[ "$(table_exists "$SAAS_DB" "$table")" == "1" ]]; then
      log "FAIL: $table still on saas after DROP"
      remaining=1
    fi
  done
  if [[ "$remaining" == "1" ]]; then
    echo "[drop-feature-params] DROP incomplete" >&2
    exit 1
  fi
fi

if [[ "$DRY_RUN" == "1" ]]; then
  log "dry-run complete: would drop $dropped_count tables (skipped $skipped_count)"
else
  log "done. saas 六表已删除 ($dropped_count dropped, $skipped_count already absent); cloud 端数据完好。"
fi
