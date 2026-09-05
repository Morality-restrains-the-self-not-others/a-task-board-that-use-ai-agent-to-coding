#!/bin/bash
set -euo pipefail

# App processes must not inherit shell HTTP(S)_PROXY (dev-only network accel).
# See .ai/01_project_constraints/23_app_startup_no_env_proxy.md
unset http_proxy https_proxy HTTP_PROXY HTTPS_PROXY ALL_PROXY all_proxy || true
REPO_ROOT="$(cd "$(dirname "$0")/.." && pwd)"
SERVICE="taskTenantService"
cd "$REPO_ROOT/$SERVICE"
echo "[$SERVICE] starting on :8020..."
exec ./bin/$SERVICE
