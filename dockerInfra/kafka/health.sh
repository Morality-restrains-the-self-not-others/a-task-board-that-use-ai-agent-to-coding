#!/usr/bin/env bash
# Exit 0 only when the Compose Kafka *broker* process answers.
# Kafka UI (:18080) alone is NOT sufficient — UI can stay up while broker Exited
# (see .ai/09_failure_experience/02_runtime_errors/70_kafka_broker_down_ui_healthy_false_positive.md).
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "$ROOT"

compose() {
  if docker compose version >/dev/null 2>&1; then
    docker compose -f docker-compose.yml "$@"
  else
    docker-compose -f docker-compose.yml "$@"
  fi
}

if ! command -v docker >/dev/null 2>&1; then
  echo "docker not found" >&2
  exit 1
fi

cid="$(compose ps -q kafka 2>/dev/null || true)"
if [[ -z "$cid" ]]; then
  echo "compose kafka broker container not running" >&2
  compose ps >&2 || true
  exit 1
fi

state="$(docker inspect -f '{{.State.Status}}' "$cid" 2>/dev/null || echo unknown)"
if [[ "$state" != "running" ]]; then
  echo "compose kafka broker not running: state=$state" >&2
  compose ps >&2 || true
  exit 1
fi

# Broker protocol probe (internal listener). Source of truth — not Kafka UI HTTP,
# and not Docker health "starting" alone (topics can succeed before health=healthy).
if ! docker exec "$cid" kafka-topics --bootstrap-server localhost:9092 --list >/dev/null 2>&1; then
  health="$(docker inspect -f '{{if .State.Health}}{{.State.Health.Status}}{{else}}none{{end}}' "$cid" 2>/dev/null || echo unknown)"
  echo "kafka broker not answering kafka-topics on localhost:9092 (docker_health=$health)" >&2
  compose ps >&2 || true
  exit 1
fi

exit 0
