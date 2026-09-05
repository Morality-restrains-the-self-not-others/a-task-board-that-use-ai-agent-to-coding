#!/bin/bash
# 编译 runAll：源码在 src/，产物在 bin/runAll
set -euo pipefail

ROOT="$(cd "$(dirname "$0")" && pwd)"
cd "$ROOT"

mkdir -p bin

echo "==> Building runAll (./src -> bin/runAll)..."
go build -buildvcs=false -o bin/runAll ./src

# Hot-replace via /api/shutdown-self MUST use a binary that skips orphan cleanup.
# Without this marker, a new instance may SIGKILL still-running managed services.
# Use byte search (not `strings|grep`): Go packs adjacent literals without NULs.
SKIP_ORPHAN_MARKER="skip orphan port cleanup"
if ! grep -aF -q "$SKIP_ORPHAN_MARKER" bin/runAll; then
  echo "ERROR: bin/runAll missing capability marker: $SKIP_ORPHAN_MARKER" >&2
  echo "Do not deploy this binary for shutdown-self hot-replace." >&2
  exit 1
fi

echo "==> Verified skip-orphan capability: $SKIP_ORPHAN_MARKER"
echo "==> Done: $ROOT/bin/runAll"
echo "==> Hot-replace tip: ./run.sh (or restart after build) — never replace with an older binary lacking skip-orphan"
