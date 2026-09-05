#!/usr/bin/env bash
set -euo pipefail
ROOT="$(cd "$(dirname "$0")" && pwd)"
cd "$ROOT"
mkdir -p bin
echo "==> Building taskGitOauth (./src -> bin/taskGitOauth)..."
go build -buildvcs=false -o bin/taskGitOauth ./src
echo "==> Done: $ROOT/bin/taskGitOauth"
