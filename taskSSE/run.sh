#!/usr/bin/env bash
set -euo pipefail

# App processes must not inherit shell HTTP(S)_PROXY (dev-only network accel).
# See .ai/01_project_constraints/23_app_startup_no_env_proxy.md
unset http_proxy https_proxy HTTP_PROXY HTTPS_PROXY ALL_PROXY all_proxy || true
cd "$(dirname "$0")"

cmd="${1:-start}"
case "$cmd" in
  start)
    if [[ ! -d node_modules ]]; then
      npm install --no-fund --no-audit
    fi
    exec node src/server.mjs
    ;;
  stop)
    pkill -f 'taskSSE.*server.mjs' 2>/dev/null || true
    pkill -f 'node src/server.mjs' 2>/dev/null || true
    lsof -ti:8798 2>/dev/null | xargs kill -9 2>/dev/null || true
    ;;
  *)
    echo "usage: $0 {start|stop}" >&2
    exit 1
    ;;
esac
