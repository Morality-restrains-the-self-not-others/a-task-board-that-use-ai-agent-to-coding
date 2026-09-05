#!/usr/bin/env bash
# Wrapper: collect Release payloads into /tmp/ram-work/deploy-binaries (gitignore).
set -euo pipefail
ROOT="$(cd "$(dirname "$0")/.." && pwd)"
exec "$ROOT/runAll/scripts/collect-deploy-binaries.sh" "$@"
