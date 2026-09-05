#!/usr/bin/env bash
# 一次性：从 Saas_project 共享库复制 taskAuth 表到 taskAuth/data/auth.db
set -euo pipefail
ROOT="$(cd "$(dirname "$0")/../.." && pwd)"
SHARED="${SHARED_DB:-$ROOT/db/saas/saas.sqlite3}"
AUTH_DB="${AUTH_DB:-$ROOT/db/task-auth/auth.sqlite3}"

mkdir -p "$(dirname "$AUTH_DB")"
rm -f "$AUTH_DB"

sqlite3 "$AUTH_DB" < "$ROOT/taskAuth/migrations/001_auth_tables.sql"
sqlite3 "$AUTH_DB" "INSERT INTO taskauth_schema_migrations (name, applied_at) VALUES ('001_auth_tables.sql', datetime('now'));"

sqlite3 "$AUTH_DB" <<SQL
ATTACH '$SHARED' AS src;
INSERT OR REPLACE INTO django_content_type SELECT * FROM src.django_content_type WHERE app_label='accounts' AND model='user';
INSERT OR REPLACE INTO accounts_user SELECT id, password, last_login, is_superuser, is_staff, is_active, date_joined FROM src.accounts_user;
INSERT OR REPLACE INTO accounts_login_method SELECT * FROM src.accounts_login_method;
INSERT OR REPLACE INTO accounts_customtoken SELECT * FROM src.accounts_customtoken;
DETACH src;
SQL

echo "Migrated auth tables to $AUTH_DB"
