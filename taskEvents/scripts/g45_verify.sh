#!/usr/bin/env bash
# G4.5 automated gate: integration (no consumers) → start events → health.
set -euo pipefail
ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT"

echo "== G4.5 verify: stop consumers (avoid redis double-consume during tests) =="
bash run.sh stop all 2>/dev/null || true
for port in $(seq 18020 18044); do
  lsof -ti:"$port" 2>/dev/null | xargs kill -9 2>/dev/null || true
done
sleep 1

echo "== integration tests (Redis only, no event binaries) =="
bash run.sh integration-test

echo "== start event-level binaries =="
bash run.sh start events
sleep 2

echo "== health check enabled intents (18020-18044) =="
sleep 5
bash scripts/check_event_health.sh

echo "G4.5 automated checks passed."
