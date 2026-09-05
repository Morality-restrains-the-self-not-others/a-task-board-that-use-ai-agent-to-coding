#!/usr/bin/env bash
set -euo pipefail
ROOT="$(cd "$(dirname "$0")/../.." && pwd)"
PYTHON="${PYTHON:-python3}"
"$PYTHON" "$ROOT/db/_infra/kafka_recreate.py"
