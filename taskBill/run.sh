#!/usr/bin/env bash
set -euo pipefail

# App processes must not inherit shell HTTP(S)_PROXY (dev-only network accel).
# See .ai/01_project_constraints/23_app_startup_no_env_proxy.md
unset http_proxy https_proxy HTTP_PROXY HTTPS_PROXY ALL_PROXY all_proxy || true
cd "$(dirname "$0")"

# 只允许从目录名恰为 taskBill 的主仓路径启动，避免旁路 worktree
# （如历史 taskBill-filter-parity）旧二进制占 :8004 且漏序列化 GitLab 价目字段。
canonical="$(pwd -P)"
base="$(basename "$canonical")"
if [[ "$base" != "taskBill" ]]; then
  echo "error: refuse to start taskBill from non-canonical directory: $canonical" >&2
  echo "hint: use /tmp/ram-work/taskBill (runAll working_dir: taskBill) or merge GitLab pricing JSON into this worktree" >&2
  exit 1
fi
if [[ "$canonical" == *filter-parity* ]]; then
  echo "error: refuse to start from filter-parity worktree: $canonical" >&2
  exit 1
fi

cmd="${1:-start}"
case "$cmd" in
  build)
    ./build.sh
    ;;
  start)
    # ADR-0027: start execs last-good only. Compile via `run.sh build` or runAll 精准编译重启.
    if [[ ! -x ./bin/taskBill ]]; then
      echo "cannot start: missing ./bin/taskBill (run: bash run.sh build)" >&2
      exit 1
    fi
    exec ./bin/taskBill
    ;;
  stop)
    # 与 runAll stop_command 一致：按端口清理，避免旁路进程残留
    lsof -ti:8004 2>/dev/null | xargs kill -9 2>/dev/null || true
    pkill -f '[/]bin/taskBill|[.]/bin/taskBill' 2>/dev/null || pkill -f '[/]taskBill' 2>/dev/null || true
    ;;
  migrate)
    if [[ -x ./bin/taskBill ]]; then
      exec ./bin/taskBill migrate
    fi
    go run ./src migrate
    ;;
  *)
    echo "usage: $0 {build|start|stop|migrate}" >&2
    exit 1
    ;;
esac
