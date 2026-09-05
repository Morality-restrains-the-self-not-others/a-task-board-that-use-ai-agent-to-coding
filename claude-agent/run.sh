#!/bin/bash
set -euo pipefail
# App processes must not inherit shell HTTP(S)_PROXY (dev-only network accel).
# See .ai/01_project_constraints/23_app_startup_no_env_proxy.md
unset http_proxy https_proxy HTTP_PROXY HTTPS_PROXY ALL_PROXY all_proxy || true

# ============================================================
# Claude Agent — Run Script (Go)
# ============================================================
SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
BINARY="$SCRIPT_DIR/bin/claude-agent"

if [ ! -f "$BINARY" ]; then
    echo "Building claude-agent..."
    "$SCRIPT_DIR/build.sh"
fi

exec "$BINARY" "$@"
