#!/usr/bin/env bash
# DROP saas 中已归档的 Budget usage *_legacy 表（不可逆）。
# 前置：已跑 archive_saas_legacy_usage.sh，且 archive/ 下有对应导出。
# 不动 projects_task_api_key_usage；不动 task_budget.db。
# 用法:
#   bash db/task_budget/drop_saas_legacy_usage.sh [--dry-run] [--force]
# 环境变量:
#   SAAS_DATABASE_PATH / ARCHIVE_DIR
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/../.." && pwd)"
SAAS_DB="${SAAS_DATABASE_PATH:-$ROOT/db/saas/saas.sqlite3}"
ARCHIVE_DIR="${ARCHIVE_DIR:-$ROOT/db/task_budget/archive}"
DRY_RUN=0
FORCE=0

for arg in "$@"; do
  case "$arg" in
    --dry-run|-n) DRY_RUN=1 ;;
    --force|-f) FORCE=1 ;;
    -h|--help)
      sed -n '2,10p' "$0"
      exit 0
      ;;
    *)
      echo "[drop-legacy-usage] unknown arg: $arg" >&2
      exit 2
      ;;
  esac
done

USAGE_LEGACY="projects_task_model_budget_usage_legacy"
IDEM_LEGACY="projects_task_model_budget_usage_idempotency_legacy"
PROTECTED_TABLE="projects_task_api_key_usage"

log() { echo "[drop-legacy-usage] $*"; }

require_cmd() {
  command -v "$1" >/dev/null 2>&1 || {
    echo "[drop-legacy-usage] missing command: $1" >&2
    exit 1
  }
}

table_exists() {
  local db="$1" table="$2"
  sqlite3 "$db" "SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name='$table';"
}

require_cmd sqlite3

if [[ ! -f "$SAAS_DB" ]]; then
  echo "[drop-legacy-usage] saas db missing: $SAAS_DB" >&2
  exit 1
fi

# 必须有至少一份 usage 归档导出（sql 或 csv）
shopt -s nullglob
USAGE_ARCHIVES=("$ARCHIVE_DIR"/projects_task_model_budget_usage.*.sql "$ARCHIVE_DIR"/projects_task_model_budget_usage.*.csv)
if [[ ${#USAGE_ARCHIVES[@]} -eq 0 ]]; then
  echo "[drop-legacy-usage] refuse: no archive export under $ARCHIVE_DIR" >&2
  exit 1
fi
log "archive exports present: ${#USAGE_ARCHIVES[@]} file(s)"

# 保护表不得被误 DROP
if [[ "$(table_exists "$SAAS_DB" "$PROTECTED_TABLE")" != "1" ]]; then
  log "warn: $PROTECTED_TABLE not on saas (expected for FeatureParams usage)"
fi

drop_one() {
  local table="$1"
  if [[ "$(table_exists "$SAAS_DB" "$table")" != "1" ]]; then
    log "idempotent: $table already absent"
    return 0
  fi
  local n
  n="$(sqlite3 "$SAAS_DB" "SELECT COUNT(*) FROM \"$table\";")"
  if [[ "$DRY_RUN" == "1" ]]; then
    log "dry-run would: DROP TABLE $table (rows=$n)"
    return 0
  fi
  if [[ "$FORCE" != "1" ]]; then
    echo "[drop-legacy-usage] refuse DROP without --force (table=$table rows=$n)" >&2
    exit 1
  fi
  sqlite3 "$SAAS_DB" "DROP TABLE \"$table\";"
  log "dropped saas.$table (was rows=$n)"
}

drop_one "$USAGE_LEGACY"
drop_one "$IDEM_LEGACY"

# 断言：保护表仍在；legacy 已不在（或 dry-run）
if [[ "$DRY_RUN" != "1" ]]; then
  if [[ "$(table_exists "$SAAS_DB" "$USAGE_LEGACY")" == "1" ]] || \
     [[ "$(table_exists "$SAAS_DB" "$IDEM_LEGACY")" == "1" ]]; then
    echo "[drop-legacy-usage] DROP incomplete" >&2
    exit 1
  fi
  if [[ "$(table_exists "$SAAS_DB" "projects_task_model_budget_usage")" == "1" ]] || \
     [[ "$(table_exists "$SAAS_DB" "projects_task_model_budget_usage_idempotency")" == "1" ]]; then
    echo "[drop-legacy-usage] unexpected non-legacy usage tables still on saas" >&2
    exit 1
  fi
fi

if [[ "$DRY_RUN" == "1" ]]; then
  log "dry-run complete (no changes)"
else
  log "done. saas *_legacy usage tables removed; task_budget.db unchanged."
fi
