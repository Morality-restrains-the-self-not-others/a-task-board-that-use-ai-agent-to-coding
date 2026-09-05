#!/usr/bin/env bash
set -euo pipefail
cd "$(dirname "$0")"
# App processes must not inherit shell HTTP(S)_PROXY (dev-only network accel).
# See .ai/01_project_constraints/23_app_startup_no_env_proxy.md
unset http_proxy https_proxy HTTP_PROXY HTTPS_PROXY ALL_PROXY all_proxy || true
cmd="${1:-start}"
case "$cmd" in
  build) ./build.sh ;;
  start)
    # ADR-0027: start execs last-good only. Compile via `run.sh build` or runAll 精准编译重启.
    if [[ ! -x ./bin/taskAiProvider ]]; then
      echo "cannot start: missing ./bin/taskAiProvider (run: bash run.sh build)" >&2
      exit 1
    fi
    # OPT-20260806-024: missing dist → SPA silently 404s; fail fast instead.
    if [ ! -f frontend/dist/index.html ]; then
      echo "ERROR: frontend/dist/index.html missing — run: bash run.sh build" >&2
      exit 1
    fi
    exec ./bin/taskAiProvider
    ;;
  stop)
    pkill -f '[/]bin/taskAiProvider|[.]/bin/taskAiProvider' 2>/dev/null || true
    # Release :8010 if a stale listener remains (runAll health binds this port).
    if command -v fuser >/dev/null 2>&1; then
      fuser -k 8010/tcp 2>/dev/null || true
    fi
    ;;
  *) echo "usage: $0 {build|start|stop}" >&2; exit 1 ;;
esac
