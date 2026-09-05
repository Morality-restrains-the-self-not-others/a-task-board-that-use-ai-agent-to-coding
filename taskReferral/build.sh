#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "$0")" && pwd)"
cd "$ROOT"

mkdir -p bin

echo "==> Building taskReferral (./src -> bin/taskReferral)..."
GONOSUMDB='*' GONOSUMCHECK='*' go build -buildvcs=false -o bin/taskReferral ./src

echo "==> Done: $ROOT/bin/taskReferral"
