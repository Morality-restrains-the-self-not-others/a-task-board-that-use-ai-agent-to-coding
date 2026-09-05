#!/usr/bin/env bash
# Exit 0 when Compose Redis is running/healthy, OR when start intentionally
# reused a host Redis (marker / DOCKER_REDIS_REUSE_HOST=1) and PING succeeds.
# Plain TCP :6379 without marker is NOT enough (avoids false healthy when
# systemd redis-server occupies the port but Compose never started).
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "$ROOT"

REUSE_MARKER="${ROOT}/.reuse_host_redis"
HOST_REDIS_HOST="${DOCKER_REDIS_HOST:-127.0.0.1}"
HOST_REDIS_PORT="${DOCKER_REDIS_PORT:-6379}"

compose() {
  if docker compose version >/dev/null 2>&1; then
    docker compose -f docker-compose.yml "$@"
  else
    docker-compose -f docker-compose.yml "$@"
  fi
}

host_redis_ping() {
  if command -v redis-cli >/dev/null 2>&1; then
    redis-cli -h "$HOST_REDIS_HOST" -p "$HOST_REDIS_PORT" ping 2>/dev/null | grep -qx PONG
    return $?
  fi
  if command -v bash >/dev/null 2>&1; then
    timeout 2 bash -c "echo >/dev/tcp/${HOST_REDIS_HOST}/${HOST_REDIS_PORT}" 2>/dev/null
    return $?
  fi
  return 1
}

reuse_allowed() {
  case "${DOCKER_REDIS_REUSE_HOST:-}" in
    1|true|TRUE|yes|YES) return 0 ;;
  esac
  [[ -f "$REUSE_MARKER" ]]
}

if ! command -v docker >/dev/null 2>&1; then
  echo "docker not found" >&2
  exit 1
fi

cid="$(compose ps -q redis 2>/dev/null || true)"
if [[ -n "$cid" ]]; then
  status="$(docker inspect -f '{{if .State.Health}}{{.State.Health.Status}}{{else}}{{.State.Status}}{{end}}' "$cid" 2>/dev/null || echo unknown)"
  case "$status" in
    healthy|running)
      exit 0
      ;;
  esac
fi

if reuse_allowed; then
  if host_redis_ping; then
    exit 0
  fi
  echo "host redis reuse enabled but PING failed at ${HOST_REDIS_HOST}:${HOST_REDIS_PORT}" >&2
  exit 1
fi

if [[ -z "$cid" ]]; then
  echo "compose redis container not running" >&2
  compose ps >&2 || true
  if host_redis_ping; then
    echo "note: host ${HOST_REDIS_HOST}:${HOST_REDIS_PORT} answers PING, but reuse is not enabled." >&2
    echo "hint: re-run bash dockerInfra/redis/run.sh start (auto-reuse on port conflict)," >&2
    echo "      or export DOCKER_REDIS_REUSE_HOST=1" >&2
  fi
  exit 1
fi

status="$(docker inspect -f '{{if .State.Health}}{{.State.Health.Status}}{{else}}{{.State.Status}}{{end}}' "$cid" 2>/dev/null || echo unknown)"
echo "compose redis unhealthy: status=$status" >&2
compose ps >&2 || true
exit 1
