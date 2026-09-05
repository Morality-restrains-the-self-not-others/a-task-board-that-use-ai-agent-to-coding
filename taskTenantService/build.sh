#!/bin/bash
set -euo pipefail
REPO_ROOT="$(cd "$(dirname "$0")/.." && pwd)"
SERVICE="taskTenantService"
cd "$REPO_ROOT/$SERVICE"
echo "[$SERVICE] checking duplicate package symbols..."
python3 "$REPO_ROOT/db/scripts/ci/check_go_duplicate_symbols.py" --src-dir ./src
echo "[$SERVICE] building..."
go build -buildvcs=false -o bin/$SERVICE ./src/
echo "[$SERVICE] build complete"
