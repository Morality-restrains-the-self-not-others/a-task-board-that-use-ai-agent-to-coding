#!/usr/bin/env bash
# Smoke: taskGateway routes check + optional Docker health (skip if compose unavailable).
set -euo pipefail
ROOT="$(cd "$(dirname "$0")/../../.." && pwd)"

bash "$(dirname "$0")/check_routes.sh"

docker_compose() {
  if docker compose version >/dev/null 2>&1; then
    docker compose "$@"
  elif command -v docker-compose >/dev/null 2>&1; then
    docker-compose "$@"
  else
    return 127
  fi
}

if ! docker_compose version >/dev/null 2>&1; then
  echo "[smoke_taskgateway] skip: docker compose not available"
  exit 0
fi

cd "$ROOT/taskGateway"
bash run.sh routes-apply
bash run.sh start
trap 'bash run.sh stop' EXIT

for i in $(seq 1 30); do
  if curl -sk --max-time 3 "https://10.2.150.89:8443/api/health/" | grep -qiE 'ok|healthy|status'; then
    echo "[smoke_taskgateway] health OK"
    exit 0
  fi
  sleep 2
done

echo "[smoke_taskgateway] health check failed" >&2
exit 1
