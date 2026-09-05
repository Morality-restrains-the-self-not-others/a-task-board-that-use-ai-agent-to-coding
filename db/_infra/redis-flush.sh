#!/usr/bin/env bash
set -euo pipefail
ROOT="$(cd "$(dirname "$0")/../.." && pwd)"

# Read Redis port from config (same logic as before)
PORT="$(
  python3 -c "
import yaml
from pathlib import Path
p = Path('$ROOT/conf/infra/docker-infra/config.yaml')
data = yaml.safe_load(p.read_text()) or {}
redis_cfg = data.get('redis', {})
print(redis_cfg.get('port', 6379))
"
)"

COMPOSE_FILE="$ROOT/dockerInfra/redis/docker-compose.yml"
REDIS_CONTAINER=""

# Resolve Redis container for docker exec.
# Try compose ps first, then fall back to name-based lookup.
if command -v docker >/dev/null 2>&1; then
  REDIS_CONTAINER=$(docker compose -f "$COMPOSE_FILE" ps -q redis 2>/dev/null || true)
  if [ -z "$REDIS_CONTAINER" ]; then
    REDIS_CONTAINER=$(docker ps --filter "name=redis" --format '{{.ID}}' 2>/dev/null | head -1 || true)
  fi
fi

flush_redis() {
  if [ -n "$REDIS_CONTAINER" ]; then
    echo "[redis-flush] docker exec $REDIS_CONTAINER redis-cli -p $PORT FLUSHALL"
    docker exec "$REDIS_CONTAINER" redis-cli -p "$PORT" FLUSHALL
  elif command -v redis-cli >/dev/null 2>&1; then
    echo "[redis-flush] host redis-cli -h 127.0.0.1 -p $PORT FLUSHALL"
    redis-cli -h 127.0.0.1 -p "$PORT" FLUSHALL
  else
    echo "[redis-flush] ERROR: neither docker exec nor redis-cli available" >&2
    exit 1
  fi
}

echo "[redis-flush] container=${REDIS_CONTAINER:-NOT FOUND}"
flush_redis
echo "[redis-flush] done"
