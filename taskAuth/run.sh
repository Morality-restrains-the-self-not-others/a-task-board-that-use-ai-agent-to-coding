#!/usr/bin/env bash
set -euo pipefail

# App processes must not inherit shell HTTP(S)_PROXY (dev-only network accel).
# See .ai/01_project_constraints/23_app_startup_no_env_proxy.md
unset http_proxy https_proxy HTTP_PROXY HTTPS_PROXY ALL_PROXY all_proxy || true
cd "$(dirname "$0")"

cmd="${1:-start}"
case "$cmd" in
  build)
    ./build.sh
    ;;
  start)
    # ADR-0027: start execs last-good only. Compile via `run.sh build` or runAll 精准编译重启.
    if [[ ! -x ./bin/taskAuth ]]; then
      echo "cannot start: missing ./bin/taskAuth (run: bash run.sh build)" >&2
      exit 1
    fi
    exec ./bin/taskAuth
    ;;
  # OPT-20260808-022: 本地 dev 登录实例——host-only cookie（SSO_COOKIE_DOMAIN=localhost →
  # Domain=.localhost）才能被 localhost:4000 的浏览器接受；生产实例（start）不带此 env
  # 保持 .daydaymoney.com。独立端口 8005（8004 为 taskBill 固定端口），Vite dev proxy 仅
  # /api/auth/、/api/accounts/ 分流过来。exec -a taskAuth-dev：改 argv[0]，避免上方 stop
  # 的 pkill -f 'bin/taskAuth|/taskAuth' 误杀。
  dev)
    ./build.sh
    SSO_COOKIE_DOMAIN=localhost TASKAUTH_PORT=8005 exec -a taskAuth-dev ./bin/taskAuth
    ;;
  stop)
    # 仅停生产实例（argv[0]=./bin/taskAuth）；dev 实例（argv[0]=taskAuth-dev）用 pkill -f taskAuth-dev 单独停
    pkill -f '[/]bin/taskAuth|[.]/bin/taskAuth' 2>/dev/null || pkill -f '[/]taskAuth' 2>/dev/null || true
    ;;
  migrate)
    if [[ -x ./bin/taskAuth ]]; then
      exec ./bin/taskAuth migrate
    fi
    go run ./src migrate
    ;;
  bootstrap-admin)
    if [[ -x ./bin/taskAuth ]]; then
      exec ./bin/taskAuth bootstrap-admin
    fi
    go run ./src bootstrap-admin
    ;;
  *)
    echo "usage: $0 {build|start|stop|migrate|bootstrap-admin}" >&2
    exit 1
    ;;
esac
