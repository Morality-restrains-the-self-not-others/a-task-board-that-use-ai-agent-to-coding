#!/usr/bin/env bash
set -euo pipefail
# conf/auth/task-auth → monorepo root is ../../..
ROOT="$(cd "$(dirname "$0")/../../.." && pwd)"
exec python3 "$ROOT/runAll/scripts/conf-sync.py" auth/task-auth
