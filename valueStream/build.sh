#!/bin/bash
# 编译 valueStream：源码在 src/，产物在 bin/valueStream
set -euo pipefail

ROOT="$(cd "$(dirname "$0")" && pwd)"
cd "$ROOT"

mkdir -p bin

echo "==> Building valueStream (./src -> bin/valueStream)..."
go build -buildvcs=false -o bin/valueStream ./src

echo "==> Done: $ROOT/bin/valueStream"
