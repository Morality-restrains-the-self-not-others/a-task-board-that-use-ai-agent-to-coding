#!/usr/bin/env bash
# Reset local AiMonitor observability data volumes (Loki, Promtail, Tempo, Prometheus).
# Preserves grafana_data. Intended for runAll dev "clear all observability" button.
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
COMPOSE_FILE="${AIMONITOR_COMPOSE_FILE:-$ROOT/docker-compose.yaml}"

# Promtail runs on Mac (scripts/runall-local-promtail.sh), not in this remote compose stack.
SERVICES=(otel-collector loki tempo prometheus)

compose() {
  if docker compose version >/dev/null 2>&1; then
    docker compose -f "$COMPOSE_FILE" "$@"
  elif docker-compose version >/dev/null 2>&1; then
    docker-compose -f "$COMPOSE_FILE" "$@"
  else
    echo "docker compose not found" >&2
    exit 1
  fi
}

if ! command -v docker >/dev/null 2>&1; then
  echo "docker not found" >&2
  exit 1
fi

volume_names() {
  for suffix in loki_data tempo_data prometheus_data; do
    docker volume ls --format '{{.Name}}' | grep -E "${suffix}$" || true
  done
}

wait_http() {
  local url="$1"
  local attempts="${2:-45}"
  local i=0
  while [[ $i -lt $attempts ]]; do
    if curl -sf "$url" >/dev/null 2>&1; then
      return 0
    fi
    i=$((i + 1))
    sleep 1
  done
  echo "timeout waiting for $url" >&2
  return 1
}

echo "Stopping observability services..."
compose stop "${SERVICES[@]}"

echo "Removing containers so data volumes can be deleted..."
compose rm -f "${SERVICES[@]}"

echo "Removing data volumes (loki/tempo/prometheus)..."
mapfile -t VOLS < <(volume_names | sort -u)
for vol in "${VOLS[@]}"; do
  if [[ -n "$vol" ]]; then
    if ! docker volume rm "$vol" >/dev/null 2>&1; then
      echo "failed to remove volume $vol (is another container still using it?)" >&2
      exit 1
    fi
  fi
done

echo "Recreating observability services..."
compose up -d --force-recreate --no-deps prometheus tempo otel-collector loki

wait_http "http://127.0.0.1:9090/-/healthy" 45
wait_http "http://127.0.0.1:3200/ready" 45
if ! wait_http "http://127.0.0.1:3100/ready" 60; then
  curl -sf -u admin:admin "http://127.0.0.1:3000/api/datasources/proxy/uid/loki/loki/api/v1/labels" >/dev/null \
    || { echo "Loki not ready after reset" >&2; exit 1; }
fi

LOCAL_PROMTAIL_RESET="${ROOT}/../runAll/scripts/runall-local-promtail.sh"
if [[ -f "$LOCAL_PROMTAIL_RESET" ]]; then
  echo "Resetting Mac local Promtail (positions volume)..."
  bash "$LOCAL_PROMTAIL_RESET" reset || echo "警告: local promtail reset 失败（可能未使用 desktop-linux）" >&2
  echo "Starting Promtail so Loki is refilled after volume wipe..."
  bash "$LOCAL_PROMTAIL_RESET" up || echo "警告: runall-local-promtail.sh up 失败" >&2
fi

echo "Observability storage reset complete."
