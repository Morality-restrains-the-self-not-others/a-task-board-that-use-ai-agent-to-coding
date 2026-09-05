#!/usr/bin/env bash
set -euo pipefail
# App processes must not inherit shell HTTP(S)_PROXY (dev-only network accel).
# See .ai/01_project_constraints/23_app_startup_no_env_proxy.md
unset http_proxy https_proxy HTTP_PROXY HTTPS_PROXY ALL_PROXY all_proxy || true

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "$ROOT"
# shellcheck source=../scripts/resolve_infra_conf.sh
source "$ROOT/../scripts/resolve_infra_conf.sh"
PORTAINER_CONF="$(resolve_infra_conf portainer)"
if [[ -f "$PORTAINER_CONF" ]]; then
  echo "[docker-portainer] conf=${PORTAINER_CONF} host=$(python3 "$ROOT/../scripts/infra_conf_host.py" "$PORTAINER_CONF")"
else
  echo "[docker-portainer] missing $PORTAINER_CONF" >&2
  exit 1
fi

COMPOSE_FILE="docker-compose.yml"

readonly -a REQUIRED_IMAGES=(
  "portainer/portainer-ce:2.27.0"
)

if ! command -v docker >/dev/null 2>&1; then
  echo "错误: 未找到 docker，请先安装 Docker Engine。" >&2
  exit 1
fi

compose() {
  if docker compose version >/dev/null 2>&1; then
    docker compose -f "$COMPOSE_FILE" "$@"
  elif docker-compose version >/dev/null 2>&1; then
    docker-compose -f "$COMPOSE_FILE" "$@"
  else
    echo "错误: 未找到 Docker Compose（需 docker compose 或 docker-compose）。" >&2
    exit 1
  fi
}

image_present() {
  docker image inspect "$1" >/dev/null 2>&1
}

ensure_images() {
  local missing=0
  for image in "${REQUIRED_IMAGES[@]}"; do
    if ! image_present "$image"; then
      missing=1
      echo "本地缺少镜像: $image"
    fi
  done
  if [[ "$missing" -eq 1 ]]; then
    echo "正在拉取缺失的 Portainer 镜像（仅一次）..."
    compose pull
  else
    echo "本地镜像已齐，跳过 pull。"
  fi
}

up_stack() {
  echo "正在启动 Portainer（不重复 pull）..."
  compose up -d --pull never --remove-orphans
}

down_stack() {
  echo "正在停止 Portainer 容器..."
  compose down --remove-orphans "$@"
}

mode="start"
if [[ $# -gt 0 ]]; then
  case "$1" in
    start|managed|stop)
      mode="$1"
      shift
      ;;
  esac
fi

case "$mode" in
  stop)
    down_stack "$@"
    ;;
  managed|start)
    ensure_images
    up_stack
    compose ps
    ;;
esac
