#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "$0")" && pwd)"
cd "$ROOT"

mkdir -p bin

echo "==> Building taskAIEndPoint (./src -> bin/taskAIEndPoint)..."
go build -buildvcs=false -o bin/taskAIEndPoint ./src

echo "==> Done: $ROOT/bin/taskAIEndPoint"
