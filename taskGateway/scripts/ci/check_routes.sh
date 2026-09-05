#!/usr/bin/env bash
# check_routes.sh — Cross-check Go service handler routes vs taskGateway routes.yaml
# OPT-20260726-013 / OPT-20260728-014: Prevent silent 404s when Go services add new API
# endpoints but the gateway routes.yaml is not updated.
#
# This script now delegates to the comprehensive Python checker that covers ALL Go services.
# For CI, run with --ci flag.
#
# Usage: bash scripts/ci/check_routes.sh [--ci]
#   --ci   Exit non-zero on missing routes (for CI pipeline)
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/../../.." && pwd)"

CI_FLAG=""
[[ "${1:-}" == "--ci" ]] && CI_FLAG="--ci"

echo "=== Go Route → APISIX Gateway Route Coverage Check ==="
echo ""

exec python3 "$SCRIPT_DIR/check_go_routes_vs_apisix.py" $CI_FLAG
