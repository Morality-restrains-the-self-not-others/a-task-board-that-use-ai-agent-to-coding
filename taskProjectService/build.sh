#!/bin/bash
set -euo pipefail
REPO_ROOT="$(cd "$(dirname "$0")/.." && pwd)"
SERVICE="taskProjectService"
cd "$REPO_ROOT/$SERVICE"
echo "[$SERVICE] building..."
go build -buildvcs=false -o bin/$SERVICE ./src/
echo "[$SERVICE] build complete"
