#!/usr/bin/env bash
# Wrapper: precise-compile registered source services into /tmp/ram-work/deploy-binaries.
set -euo pipefail
ROOT="$(cd "$(dirname "$0")/.." && pwd)"
exec "$ROOT/runAll/scripts/precise-compile.sh" "$@"
