#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_DIR="$(cd "${SCRIPT_DIR}/.." && pwd)"

export GRAFANA_URL="${GRAFANA_URL:-http://127.0.0.1:3000}"

cd "${PROJECT_DIR}"
npm run e2e -- tests/loginSessionPersistence.spec.ts "$@"
