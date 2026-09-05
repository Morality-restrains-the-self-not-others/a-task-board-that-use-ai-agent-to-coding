#!/usr/bin/env bash
set -euo pipefail

# App processes must not inherit shell HTTP(S)_PROXY (dev-only network accel).
# See .ai/01_project_constraints/23_app_startup_no_env_proxy.md
unset http_proxy https_proxy HTTP_PROXY HTTPS_PROXY ALL_PROXY all_proxy || true
cd "$(dirname "$0")"
cmd="${1:-start}"
case "$cmd" in
  build) ./build.sh ;;
  start)
    # ADR-0027: start execs last-good only. Compile via `run.sh build` or runAll 精准编译重启.
    if [[ ! -x ./bin/taskGitOauth ]]; then
      echo "cannot start: missing ./bin/taskGitOauth (run: bash run.sh build)" >&2
      exit 1
    fi
    exec ./bin/taskGitOauth
    ;;
  stop) pkill -f '[/]bin/taskGitOauth|[.]/bin/taskGitOauth' 2>/dev/null || true ;;
  *) echo "usage: $0 {build|start|stop}" >&2; exit 1 ;;
esac
