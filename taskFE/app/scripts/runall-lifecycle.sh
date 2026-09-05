#!/usr/bin/env bash
set -euo pipefail

# App processes must not inherit shell HTTP(S)_PROXY (dev-only network accel).
unset http_proxy https_proxy HTTP_PROXY HTTPS_PROXY ALL_PROXY all_proxy || true

# scripts/ → app/ → taskFE/ → monorepo root
script_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
app_dir="$(cd "$script_dir/.." && pwd)"
taskfe_dir="$(cd "$app_dir/.." && pwd)"
repo_root="$(cd "$taskfe_dir/.." && pwd)"
cd "$app_dir"

conf_read() {
  python3 "$repo_root/runAll/scripts/conf-read.py" vue "$1"
}

export_compose_env() {
  TASKFE_NGINX_IMAGE="$(conf_read nginxImage)"
  TASKFE_NGINX_HOST="$(conf_read host)"
  TASKFE_NGINX_PORT="$(conf_read port)"
  TASKFE_NGINX_MEM_LIMIT="$(conf_read memLimit)"
  TASKFE_RELEASE_KEEP="$(conf_read releaseKeep)"
  export TASKFE_NGINX_IMAGE TASKFE_NGINX_HOST TASKFE_NGINX_PORT TASKFE_NGINX_MEM_LIMIT TASKFE_RELEASE_KEEP
}

compose() {
  docker compose -f "$taskfe_dir/docker-compose.yml" -p taskfe "$@"
}

# 切换期：:4000 上遗留的 node/vite preview 会挡住 docker-proxy。对其 SIGTERM，不用强制 SIGKILL。
release_stale_node_on_port() {
  local port="${TASKFE_NGINX_PORT:-4000}"
  local pid comm
  command -v lsof >/dev/null 2>&1 || return 0
  while read -r pid; do
    [[ -n "$pid" ]] || continue
    comm="$(ps -p "$pid" -o comm= 2>/dev/null || true)"
    case "$comm" in
      docker-proxy|nginx|nginx:*) continue ;;
    esac
    echo "runall-lifecycle: SIGTERM leftover pid=$pid comm=$comm on :${port} (vite preview 不再作为公网入口)" >&2
    kill "$pid" 2>/dev/null || true
  done < <(lsof -tiTCP:"$port" -sTCP:LISTEN 2>/dev/null || true)
  sleep 0.3
}

ensure_html() {
  mkdir -p public
  if [ -f public/html/index.html ]; then
    return 0
  fi
  if [ -f dist/index.html ]; then
    mkdir -p public/releases/migrated-from-dist
    cp -a dist/. public/releases/migrated-from-dist/
    ln -sfn releases/migrated-from-dist public/html.tmp
    mv -T public/html.tmp public/html
    echo "runall-lifecycle: migrated dist/ → public/html" >&2
  fi
  if [ ! -f public/html/index.html ]; then
    echo "public/html/index.html 缺失：请先 bash scripts/runall-lifecycle.sh build 再 start" >&2
    exit 1
  fi
}

cmd="${1:-start}"
case "$cmd" in
  start)
    export_compose_env
    # public/html/index.html 必须存在（ensure_html；无产物禁止启动）
    ensure_html
    release_stale_node_on_port
    compose up -d --wait
    echo "runall-lifecycle: taskfe-nginx up ${TASKFE_NGINX_HOST}:${TASKFE_NGINX_PORT}"
    ;;
  stop)
    export_compose_env
    compose stop
    ;;
  build)
    export_compose_env
    exec bash "$script_dir/atomic-vite-build.sh"
    ;;
  *)
    echo "usage: $0 {start|stop|build}" >&2
    exit 1
    ;;
esac
