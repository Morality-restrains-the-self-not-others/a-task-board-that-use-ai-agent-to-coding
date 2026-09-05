#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "$0")" && pwd)"
cd "$ROOT"

mkdir -p bin

echo "==> Building taskAgentSupport (./src -> bin/taskAgentSupport)..."
go build -buildvcs=false -o bin/taskAgentSupport ./src

echo "==> Done: $ROOT/bin/taskAgentSupport"
