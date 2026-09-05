#!/usr/bin/env bash
# 编译 go_relayToTrae：源码在 src/，产物在 bin/go_relayToTrae
set -euo pipefail

ROOT="$(cd "$(dirname "$0")" && pwd)"
cd "$ROOT"

mkdir -p bin

echo "==> Building go_relayToTrae (./src -> bin/go_relayToTrae)..."
go build -tags otel_enabled -buildvcs=false -o bin/go_relayToTrae ./src

echo "==> Done: $ROOT/bin/go_relayToTrae"
