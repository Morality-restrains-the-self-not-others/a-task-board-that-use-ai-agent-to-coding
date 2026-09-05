#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "$0")" && pwd)"
cd "$ROOT"

mkdir -p bin

echo "==> Building taskAuth (./src -> bin/taskAuth)..."
go build -tags otel_enabled -buildvcs=false -o bin/taskAuth ./src

echo "==> Done: $ROOT/bin/taskAuth"
