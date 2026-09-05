#!/usr/bin/env bash
# check_inline_ddl.sh — CI static check for inline DDL in Go application code.
#
# Detects CREATE TABLE statements embedded in Go source files (not in dataMigrate/).
# Only data_migrate_log creation is exempt (it IS the migration mechanism itself).
# Table-rebuild temp tables (__new suffix) and integration test fixtures are also exempt.
# ALTER TABLE statements in legacy-compatibility code paths are noted as warnings.
#
# Usage: bash dataMigrate/check_inline_ddl.sh [repo_root]
# Exit: 0 = clean, 1 = CREATE TABLE violations found

set -euo pipefail
ROOT="${1:-$(cd "$(dirname "$0")/.." && pwd)}"

EXEMPT_PATTERN="data_migrate_log"
CREATE_COUNT=0
ALTER_COUNT=0

echo "=== dataMigrate Inline DDL Check ==="
echo ""

# Search for CREATE TABLE in Go files (excluding test files, dataMigrate/, vendor/)
while IFS=: read -r file line content; do
    # Skip exempt pattern (data_migrate_log is the migration mechanism itself)
    if echo "$content" | grep -q "$EXEMPT_PATTERN"; then
        continue
    fi
    # Skip comments
    trimmed=$(echo "$content" | sed 's/^[[:space:]]*//')
    if echo "$trimmed" | grep -qE '^\s*//'; then
        continue
    fi
    # Skip table rebuild temp tables (e.g. comments__new — migration helper, not new schema)
    if echo "$content" | grep -qE 'CREATE TABLE.*__new'; then
        continue
    fi
    # Skip integration test fixtures (non-production)
    if echo "$file" | grep -q 'integration/fixture'; then
        continue
    fi
    # Skip table rebuild functions that DROP before CREATE (migration helper, not new schema)
    if echo "$content" | grep -qE 'CREATE TABLE[^)]*tenant_installed_images'; then
        if grep -q 'DROP TABLE.*tenant_installed_images' "$file"; then
            continue
        fi
    fi
    # Skip taskAuth delivery_attempt (registered as Go dataMigrate step)
    if echo "$file" | grep -q 'auth_email_invite.go'; then
        continue
    fi
    echo "  CREATE TABLE: $file:$line"
    CREATE_COUNT=$((CREATE_COUNT + 1))
done < <(
    find "$ROOT" \
        -type f -name "*.go" \
        ! -path "*/dataMigrate/*" \
        ! -path "*/.git/*" \
        ! -path "*/vendor/*" \
        ! -path "*_test.go" \
        ! -path "*/node_modules/*" \
        ! -path "*/dockerInfra/*" \
        ! -path "*/trae-agent/*" \
        ! -path "*/sdk/*" \
        -exec grep -Hn 'CREATE TABLE' {} \; 2>/dev/null || true
)

# Search for ALTER TABLE in Go files (informational only — legacy compatibility)
while IFS=: read -r file line content; do
    trimmed=$(echo "$content" | sed 's/^[[:space:]]*//')
    if echo "$trimmed" | grep -qE '^\s*//'; then
        continue
    fi
    ALTER_COUNT=$((ALTER_COUNT + 1))
done < <(
    find "$ROOT" \
        -type f -name "*.go" \
        ! -path "*/dataMigrate/*" \
        ! -path "*/.git/*" \
        ! -path "*/vendor/*" \
        ! -path "*_test.go" \
        ! -path "*/node_modules/*" \
        ! -path "*/dockerInfra/*" \
        ! -path "*/trae-agent/*" \
        ! -path "*/sdk/*" \
        -exec grep -Hn 'ALTER TABLE' {} \; 2>/dev/null || true
)

echo ""
echo "CREATE TABLE violations: $CREATE_COUNT"
echo "ALTER TABLE  (legacy):   $ALTER_COUNT"

if [ "$CREATE_COUNT" -gt 0 ]; then
    echo ""
    echo "FAIL: $CREATE_COUNT CREATE TABLE statement(s) embedded in application code."
    echo "Move them to dataMigrate/<service>/NNN_schema.sql files."
    echo "See: .cursor/rules/database-schema-centralized-migration.mdc"
    exit 1
fi

if [ "$ALTER_COUNT" -gt 0 ]; then
    echo ""
    echo "NOTE: $ALTER_COUNT ALTER TABLE statement(s) found. These are likely legacy"
    echo "column additions for existing databases. Consider migrating to dataMigrate"
    echo "when the column becomes part of the standard schema."
fi

echo "PASS: No inline CREATE TABLE violations."
exit 0
