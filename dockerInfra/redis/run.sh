#!/usr/bin/env bash
set -euo pipefail
# App processes must not inherit shell HTTP(S)_PROXY (dev-only network accel).
# See .ai/01_project_constraints/23_app_startup_no_env_proxy.md
unset http_proxy https_proxy HTTP_PROXY HTTPS_PROXY ALL_PROXY all_proxy || true

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "$ROOT"
# shellcheck source=../scripts/resolve_infra_conf.sh
source "$ROOT/../scripts/resolve_infra_conf.sh"
REDIS_CONF="$(resolve_infra_conf redis)"
if [[ -f "$REDIS_CONF" ]]; then
  echo "[docker-redis] conf=${REDIS_CONF} host=$(python3 "$ROOT/../scripts/infra_conf_host.py" "$REDIS_CONF")"
else
  echo "[docker-redis] missing $REDIS_CONF" >&2
  exit 1
fi

COMPOSE_FILE="docker-compose.yml"
REUSE_MARKER="${ROOT}/.reuse_host_redis"
HOST_REDIS_HOST="${DOCKER_REDIS_HOST:-127.0.0.1}"
HOST_REDIS_PORT="${DOCKER_REDIS_PORT:-6379}"

readonly -a REQUIRED_IMAGES=(
  "redis:7.0-alpine"
)

if ! command -v docker >/dev/null 2>&1; then
  echo "错误: 未找到 docker，请先安装 Docker Desktop 或 Docker Engine。" >&2
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
    echo "正在拉取缺失的 Redis 镜像（仅一次）..."
    compose pull
  else
    echo "本地镜像已齐，跳过 pull。"
  fi
}

compose_redis_cid() {
  compose ps -q redis 2>/dev/null || true
}

compose_redis_running() {
  local cid status
  cid="$(compose_redis_cid)"
  [[ -n "$cid" ]] || return 1
  status="$(docker inspect -f '{{.State.Status}}' "$cid" 2>/dev/null || echo unknown)"
  [[ "$status" == "running" ]]
}

host_redis_ping() {
  if command -v redis-cli >/dev/null 2>&1; then
    redis-cli -h "$HOST_REDIS_HOST" -p "$HOST_REDIS_PORT" ping 2>/dev/null | grep -qx PONG
    return $?
  fi
  # Fallback without redis-cli: TCP connect only (weaker, but unblocks environments
  # where redis-cli is absent and a listener is already serving the port).
  if command -v bash >/dev/null 2>&1; then
    timeout 2 bash -c "echo >/dev/tcp/${HOST_REDIS_HOST}/${HOST_REDIS_PORT}" 2>/dev/null
    return $?
  fi
  return 1
}

host_port_busy() {
  if command -v ss >/dev/null 2>&1; then
    ss -ltn "( sport = :${HOST_REDIS_PORT} )" 2>/dev/null | grep -q LISTEN
    return $?
  fi
  if command -v lsof >/dev/null 2>&1; then
    lsof -nP -iTCP:"${HOST_REDIS_PORT}" -sTCP:LISTEN >/dev/null 2>&1
    return $?
  fi
  # Unknown tooling: treat as busy only if ping succeeds.
  host_redis_ping
}

enable_host_reuse() {
  local reason="$1"
  printf '%s\n' "$reason" >"$REUSE_MARKER"
  echo "[docker-redis] reusing host Redis at ${HOST_REDIS_HOST}:${HOST_REDIS_PORT} (${reason})"
  echo "[docker-redis] marker: ${REUSE_MARKER}"
  echo "[docker-redis] tip: stop host redis-server to force Compose, or set DOCKER_REDIS_REUSE_HOST=0"
}

clear_host_reuse() {
  rm -f "$REUSE_MARKER"
}

cleanup_stale_compose() {
  # Remove Created/Exited leftovers that block a later bind attempt.
  compose rm -f redis >/dev/null 2>&1 || true
}

reuse_forced_off() {
  case "${DOCKER_REDIS_REUSE_HOST:-}" in
    0|false|FALSE|no|NO) return 0 ;;
    *) return 1 ;;
  esac
}

reuse_forced_on() {
  case "${DOCKER_REDIS_REUSE_HOST:-}" in
    1|true|TRUE|yes|YES) return 0 ;;
    *) return 1 ;;
  esac
}

try_host_reuse() {
  local reason="$1"
  if reuse_forced_off; then
    return 1
  fi
  if host_redis_ping; then
    enable_host_reuse "$reason"
    return 0
  fi
  return 1
}

up_stack() {
  if compose_redis_running; then
    clear_host_reuse
    echo "[docker-redis] Compose Redis already running."
    return 0
  fi

  if reuse_forced_on; then
    if host_redis_ping; then
      cleanup_stale_compose
      enable_host_reuse "DOCKER_REDIS_REUSE_HOST=1"
      return 0
    fi
    echo "错误: DOCKER_REDIS_REUSE_HOST=1 但 ${HOST_REDIS_HOST}:${HOST_REDIS_PORT} 无可用 Redis (PING 失败)。" >&2
    exit 1
  fi

  if ! compose_redis_running && host_port_busy; then
    if try_host_reuse "host port ${HOST_REDIS_PORT} already in use"; then
      cleanup_stale_compose
      return 0
    fi
    if reuse_forced_off; then
      echo "错误: DOCKER_REDIS_REUSE_HOST=0 要求 Compose Redis，但宿主端口 ${HOST_REDIS_PORT} 已被占用。" >&2
      echo "请先释放端口（例如: sudo systemctl stop redis-server），或取消该环境变量以允许自动复用。" >&2
      exit 1
    fi
    echo "错误: 主机端口 ${HOST_REDIS_PORT} 已被占用，且无法 PING Redis。" >&2
    echo "请释放端口后重试，例如: sudo systemctl stop redis-server" >&2
    echo "或确保占用方是可用 Redis 后设置 DOCKER_REDIS_REUSE_HOST=1。" >&2
    exit 1
  fi

  clear_host_reuse
  echo "正在启动 Redis（不重复 pull）..."
  if ! compose up -d --pull never --remove-orphans; then
    if try_host_reuse "compose up failed; host Redis reachable"; then
      cleanup_stale_compose
      return 0
    fi
    echo "错误: Compose Redis 启动失败，且主机 ${HOST_REDIS_HOST}:${HOST_REDIS_PORT} 也不可用。" >&2
    echo "常见原因: address already in use — 请检查: ss -ltnp | rg ':${HOST_REDIS_PORT}\\b'" >&2
    exit 1
  fi
}

down_stack() {
  clear_host_reuse
  echo "正在停止 Redis 容器..."
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
    if [[ -f "$REUSE_MARKER" ]]; then
      echo "[docker-redis] mode=host-reuse"
    else
      compose ps
    fi
    ;;
esac
