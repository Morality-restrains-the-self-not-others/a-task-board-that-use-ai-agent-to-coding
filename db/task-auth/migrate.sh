#!/usr/bin/env bash
# Apply dataMigrate SQL + Go seed steps for task_auth.
# Called by runAll /api/dev/init-databases (port 9999).
# Business taskAuth server does NOT run migrations on startup.
set -euo pipefail
ROOT="$(cd "$(dirname "$0")/../.." && pwd)"
bash "$ROOT/db/scripts/apply_datamigrate.sh" "task_auth" "$ROOT/dataMigrate/taskAuth"
# Go OIDC seed 每次执行、表级幂等（ensureOidcClient INSERT-if-missing），不以 step_key 跳过。
# ADR-0052: 部署根无源码，优先 last-good ELF（$DEPLOY_ROOT/bin/taskAuth）。
if [[ -x "$ROOT/bin/taskAuth" ]]; then
  exec "$ROOT/bin/taskAuth" migrate
fi
cd "$ROOT/taskAuth"
exec go run ./src migrate
