#!/usr/bin/env bash
set -euo pipefail
# App processes must not inherit shell HTTP(S)_PROXY (dev-only network accel).
# See .ai/01_project_constraints/23_app_startup_no_env_proxy.md
unset http_proxy https_proxy HTTP_PROXY HTTPS_PROXY ALL_PROXY all_proxy || true

ROOT="$(cd "$(dirname "$0")" && pwd)"
cd "$ROOT"

# ADR-0027: start execs last-good only. Compile via ./build.sh or runAll 精准编译重启.
if [[ ! -x "$ROOT/bin/taskAIEndPoint" ]]; then
  echo "cannot start: missing $ROOT/bin/taskAIEndPoint (run: $ROOT/build.sh)" >&2
  exit 1
fi

exec "$ROOT/bin/taskAIEndPoint"
