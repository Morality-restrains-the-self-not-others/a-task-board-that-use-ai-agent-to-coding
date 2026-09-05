#!/usr/bin/env bash
# Drop accounts_company from saas.sqlite3 after migrate_companies_to_task_tenant.sh.
# Child tables keep company_id values; FK enforcement disabled during drop (SQLite).
set -euo pipefail
REPO_ROOT="$(cd "$(dirname "$0")/../.." && pwd)"
SAAS_DB="${REPO_ROOT}/db/saas/saas.sqlite3"
TENANT_DB="${REPO_ROOT}/data/task_tenant.db"

if [[ ! -f "$SAAS_DB" ]]; then
  echo "missing $SAAS_DB" >&2
  exit 1
fi
if [[ ! -f "$TENANT_DB" ]]; then
  echo "missing $TENANT_DB — run migrate_companies_to_task_tenant.sh first" >&2
  exit 1
fi

saas_n=$(sqlite3 "$SAAS_DB" "SELECT count(*) FROM accounts_company;" 2>/dev/null || echo 0)
tenant_n=$(sqlite3 "$TENANT_DB" "SELECT count(*) FROM accounts_company;")
echo "saas_companies=$saas_n tenant_companies=$tenant_n"
if [[ "$saas_n" -gt 0 && "$tenant_n" -lt "$saas_n" ]]; then
  echo "refusing DROP: tenant DB has fewer companies than saas ($tenant_n < $saas_n)" >&2
  exit 1
fi

BACKUP="${SAAS_DB}.company-pre-drop.$(date +%Y%m%d%H%M%S).sql"
sqlite3 "$SAAS_DB" <<SQL >"$BACKUP"
.mode insert accounts_company
SELECT * FROM accounts_company;
SQL
echo "backup $BACKUP"

sqlite3 "$SAAS_DB" <<'SQL'
PRAGMA foreign_keys=OFF;
DROP TABLE IF EXISTS accounts_company;
PRAGMA foreign_keys=ON;
SQL

echo "dropped accounts_company from saas"
if sqlite3 "$SAAS_DB" ".tables" | tr ' ' '\n' | rg -q '^accounts_company$'; then
  echo "table still present" >&2
  exit 1
fi
# Child tables may still declare REFERENCES accounts_company in CREATE TABLE.
python3 "${REPO_ROOT}/db/scripts/strip_saas_accounts_company_fk.py" "$SAAS_DB"
echo "ok: accounts_company absent from saas"
