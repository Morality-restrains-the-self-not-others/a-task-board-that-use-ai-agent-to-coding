#!/usr/bin/env bash
# Drop people tables from saas.sqlite3 after migrate_people_tables_to_task_tenant.sh.
# Requires taskTenantService data already verified (non-empty members when source had rows).
set -euo pipefail
REPO_ROOT="$(cd "$(dirname "$0")/../.." && pwd)"
SAAS_DB="${REPO_ROOT}/db/saas/saas.sqlite3"
TENANT_DB="${REPO_ROOT}/data/task_tenant.db"

if [[ ! -f "$SAAS_DB" ]]; then
  echo "missing $SAAS_DB" >&2
  exit 1
fi
if [[ ! -f "$TENANT_DB" ]]; then
  echo "missing $TENANT_DB — run migrate_people_tables_to_task_tenant.sh first" >&2
  exit 1
fi

saas_n=$(sqlite3 "$SAAS_DB" "SELECT count(*) FROM accounts_company_member;" 2>/dev/null || echo 0)
tenant_n=$(sqlite3 "$TENANT_DB" "SELECT count(*) FROM accounts_company_member;")
echo "saas_members=$saas_n tenant_members=$tenant_n"
if [[ "$saas_n" -gt 0 && "$tenant_n" -lt "$saas_n" ]]; then
  echo "refusing DROP: tenant DB has fewer members than saas ($tenant_n < $saas_n)" >&2
  exit 1
fi

# Backup schema+data snapshot before drop
BACKUP="${SAAS_DB}.people-pre-drop.$(date +%Y%m%d%H%M%S).sql"
sqlite3 "$SAAS_DB" <<SQL >"$BACKUP"
.mode insert accounts_company_member
SELECT * FROM accounts_company_member;
.mode insert accounts_invitation
SELECT * FROM accounts_invitation;
.mode insert accounts_company_group
SELECT * FROM accounts_company_group;
.mode insert accounts_company_group_member
SELECT * FROM accounts_company_group_member;
SQL
echo "backup $BACKUP"

sqlite3 "$SAAS_DB" <<'SQL'
PRAGMA foreign_keys=OFF;
DROP TABLE IF EXISTS accounts_company_group_member;
DROP TABLE IF EXISTS accounts_company_group;
DROP TABLE IF EXISTS accounts_invitation;
DROP TABLE IF EXISTS accounts_company_member;
PRAGMA foreign_keys=ON;
SQL

echo "dropped people tables from saas"
sqlite3 "$SAAS_DB" ".tables" | tr ' ' '\n' | rg 'accounts_company_member|accounts_invitation|accounts_company_group' && {
  echo "tables still present" >&2
  exit 1
} || echo "ok: people tables absent from saas"
