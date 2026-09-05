#!/usr/bin/env bash
# 归档 saas 中遗留 Budget usage 表，再 RENAME 为 _legacy_*（不 DROP）。
# 不动 projects_task_api_key_usage。
# 用法:
#   bash db/task_budget/archive_saas_legacy_usage.sh [--dry-run]
# 环境变量:
#   SAAS_DATABASE_PATH / TASK_BUDGET_DATABASE_PATH
#   ARCHIVE_DIR（默认 db/task_budget/archive）
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/../.." && pwd)"
SAAS_DB="${SAAS_DATABASE_PATH:-$ROOT/db/saas/saas.sqlite3}"
BUDGET_DB="${TASK_BUDGET_DATABASE_PATH:-$ROOT/db/task_budget/task_budget.db}"
ARCHIVE_DIR="${ARCHIVE_DIR:-$ROOT/db/task_budget/archive}"
DRY_RUN=0

for arg in "$@"; do
  case "$arg" in
    --dry-run|-n) DRY_RUN=1 ;;
    -h|--help)
      sed -n '2,10p' "$0"
      exit 0
      ;;
    *)
      echo "[archive-legacy-usage] unknown arg: $arg" >&2
      exit 2
      ;;
  esac
done

USAGE_TABLE="projects_task_model_budget_usage"
IDEM_TABLE="projects_task_model_budget_usage_idempotency"
USAGE_LEGACY="${USAGE_TABLE}_legacy"
IDEM_LEGACY="${IDEM_TABLE}_legacy"
# 明确保护：不得改动
PROTECTED_TABLE="projects_task_api_key_usage"

log() { echo "[archive-legacy-usage] $*"; }

require_cmd() {
  command -v "$1" >/dev/null 2>&1 || {
    echo "[archive-legacy-usage] missing command: $1" >&2
    exit 1
  }
}

table_exists() {
  local db="$1" table="$2"
  sqlite3 "$db" "SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name='$table';"
}

require_cmd sqlite3

if [[ ! -f "$SAAS_DB" ]]; then
  echo "[archive-legacy-usage] saas db missing: $SAAS_DB" >&2
  exit 1
fi
if [[ ! -f "$BUDGET_DB" ]]; then
  echo "[archive-legacy-usage] task_budget db missing: $BUDGET_DB" >&2
  exit 1
fi

# --- 校验 task_budget.db 已有目标表 ---
for t in "$USAGE_TABLE" "$IDEM_TABLE"; do
  if [[ "$(table_exists "$BUDGET_DB" "$t")" != "1" ]]; then
    echo "[archive-legacy-usage] task_budget missing table: $t (run migrate_from_saas.sh first)" >&2
    exit 1
  fi
done
log "task_budget.db has $USAGE_TABLE + $IDEM_TABLE OK"

# --- 保护表必须仍在 saas（若存在则仅校验名未被误改）---
if [[ "$(table_exists "$SAAS_DB" "$PROTECTED_TABLE")" == "1" ]]; then
  log "protected table present (untouched): $PROTECTED_TABLE"
fi

STAMP="$(date -u +%Y%m%dT%H%M%SZ)"
mkdir -p "$ARCHIVE_DIR"

export_one() {
  local table="$1"
  local out_sql="$ARCHIVE_DIR/${table}.${STAMP}.sql"
  local out_csv="$ARCHIVE_DIR/${table}.${STAMP}.csv"
  local exists
  exists="$(table_exists "$SAAS_DB" "$table")"
  if [[ "$exists" != "1" ]]; then
    # 已归档：允许从 _legacy 再导出（幂等重跑）
    local legacy="${table}_legacy"
    if [[ "$(table_exists "$SAAS_DB" "$legacy")" == "1" ]]; then
      log "saas.$table already renamed; export from $legacy"
      table="$legacy"
    else
      log "skip export: neither $table nor ${table}_legacy on saas"
      return 0
    fi
  fi
  if [[ "$DRY_RUN" == "1" ]]; then
    local n
    n="$(sqlite3 "$SAAS_DB" "SELECT COUNT(*) FROM \"$table\";")"
    log "dry-run would dump $table rows=$n → $out_sql / $out_csv"
    return 0
  fi
  sqlite3 "$SAAS_DB" ".mode insert $table" "SELECT * FROM \"$table\";" >"$out_sql"
  sqlite3 "$SAAS_DB" -header -csv "SELECT * FROM \"$table\";" >"$out_csv"
  local n
  n="$(sqlite3 "$SAAS_DB" "SELECT COUNT(*) FROM \"$table\";")"
  log "exported $table rows=$n → $out_sql , $out_csv"
}

rename_to_legacy() {
  local table="$1"
  local legacy="${table}_legacy"
  if [[ "$(table_exists "$SAAS_DB" "$table")" != "1" ]]; then
    if [[ "$(table_exists "$SAAS_DB" "$legacy")" == "1" ]]; then
      log "idempotent: $table already → $legacy"
      return 0
    fi
    log "skip rename: $table not on saas"
    return 0
  fi
  if [[ "$(table_exists "$SAAS_DB" "$legacy")" == "1" ]]; then
    echo "[archive-legacy-usage] both $table and $legacy exist; refuse rename" >&2
    exit 1
  fi
  if [[ "$DRY_RUN" == "1" ]]; then
    log "dry-run would: ALTER TABLE $table RENAME TO $legacy"
    return 0
  fi
  sqlite3 "$SAAS_DB" "ALTER TABLE \"$table\" RENAME TO \"$legacy\";"
  log "renamed saas.$table → $legacy"
}

export_one "$USAGE_TABLE"
export_one "$IDEM_TABLE"
rename_to_legacy "$USAGE_TABLE"
rename_to_legacy "$IDEM_TABLE"

# 最终断言：保护表仍在；旧名已不在（或 dry-run）
if [[ "$DRY_RUN" != "1" ]]; then
  if [[ "$(table_exists "$SAAS_DB" "$USAGE_TABLE")" == "1" ]] || \
     [[ "$(table_exists "$SAAS_DB" "$IDEM_TABLE")" == "1" ]]; then
    echo "[archive-legacy-usage] rename incomplete" >&2
    exit 1
  fi
  if [[ "$(table_exists "$SAAS_DB" "$PROTECTED_TABLE")" != "1" ]]; then
    # 保护表可能本就不存在于极简库；仅当曾存在时才强制。此处若缺失只警告。
    log "warn: $PROTECTED_TABLE not found on saas (left untouched if absent)"
  fi
fi

if [[ "$DRY_RUN" == "1" ]]; then
  log "dry-run complete (no changes)"
else
  log "done. optional DROP: bash db/task_budget/drop_saas_legacy_usage.sh --force"
fi
