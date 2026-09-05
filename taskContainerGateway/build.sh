#!/usr/bin/env bash
set -euo pipefail
ROOT="$(cd "$(dirname "$0")" && pwd)"
cd "$ROOT"
mkdir -p bin
go build -buildvcs=false -o bin/taskContainerGateway ./src
echo "==> Done: $ROOT/bin/taskContainerGateway"
