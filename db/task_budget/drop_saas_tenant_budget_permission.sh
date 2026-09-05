#!/usr/bin/env bash
# DROP saas 中已迁出的 projects_tenant_budget_permission（不可逆）。
# 前置：task_budget.db 已有同名表（Cloud schema），且 saas 行已迁完或确认无数据。
# 用法:
#   bash db/task_budget/drop_saas_tenant_budget_permission.sh [--dry-run] [--force]
# 环境变量:
#   SAAS_DATABASE_PATH / TASK_BUDGET_DATABASE_PATH / ARCHIVE_DIR
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/../.." && pwd)"
SAAS_DB="${SAAS_DATABASE_PATH:-$ROOT/db/saas/saas.sqlite3}"
BUDGET_DB="${TASK_BUDGET_DATABASE_PATH:-$ROOT/db/task_budget/task_budget.db}"
ARCHIVE_DIR="${ARCHIVE_DIR:-$ROOT/db/task_budget/archive}"
TABLE="projects_tenant_budget_permission"
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
      echo "[drop-saas-perm] unknown arg: $arg" >&2
      exit 2
      ;;
  esac
done

log() { echo "[drop-saas-perm] $*"; }

require_cmd() {
  command -v "$1" >/dev/null 2>&1 || {
    echo "[drop-saas-perm] missing command: $1" >&2
    exit 2
  }
}

require_cmd sqlite3
require_cmd mkdir

if [[ ! -f "$SAAS_DB" ]]; then
  log "saas db missing: $SAAS_DB"
  exit 1
fi

saas_count="$(sqlite3 "$SAAS_DB" "SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name='$TABLE';")"
if [[ "$saas_count" != "1" ]]; then
  log "table $TABLE not in saas — nothing to drop"
  exit 0
fi

row_count="$(sqlite3 "$SAAS_DB" "SELECT COUNT(*) FROM $TABLE;")"
log "saas.$TABLE rows=$row_count"

# Ensure budget DB has the destination table (Cloud owns it).
if [[ -f "$BUDGET_DB" ]]; then
  budget_has="$(sqlite3 "$BUDGET_DB" "SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name='$TABLE';")"
  if [[ "$budget_has" != "1" ]]; then
    log "creating $TABLE on budget db (empty schema)"
    if [[ "$DRY_RUN" -eq 0 ]]; then
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
    fi
  fi
  budget_rows="$(sqlite3 "$BUDGET_DB" "SELECT COUNT(*) FROM $TABLE;" 2>/dev/null || echo 0)"
  log "budget.$TABLE rows=$budget_rows"
else
  log "budget db missing: $BUDGET_DB (will still drop saas if --force and rows=0)"
  budget_rows=0
fi

if [[ "$row_count" != "0" ]]; then
  if [[ "$FORCE" -ne 1 ]]; then
    log "saas still has $row_count rows — refuse to DROP without --force"
    log "hint: start Cloud once to migrate, or copy rows into budget db first"
    exit 1
  fi
  mkdir -p "$ARCHIVE_DIR"
  stamp="$(date -u +%Y%m%dT%H%M%SZ)"
  archive="$ARCHIVE_DIR/${TABLE}_${stamp}.sql"
  log "archiving saas rows → $archive"
  if [[ "$DRY_RUN" -eq 0 ]]; then
    sqlite3 "$SAAS_DB" ".dump $TABLE" >"$archive"
  fi
fi

if [[ "$DRY_RUN" -eq 1 ]]; then
  log "dry-run: would DROP TABLE $TABLE from $SAAS_DB"
  exit 0
fi

sqlite3 "$SAAS_DB" "DROP TABLE IF EXISTS $TABLE;"
log "dropped saas.$TABLE"
