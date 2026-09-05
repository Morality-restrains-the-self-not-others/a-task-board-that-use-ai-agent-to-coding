#!/bin/bash
set -euo pipefail
REPO_ROOT="$(cd "$(dirname "$0")/.." && pwd)"
SERVICE="taskTaskService"
cd "$REPO_ROOT/$SERVICE"
echo "[$SERVICE] building..."
# Parent meta-repo git can be bare/broken; do not fail the binary on VCS stamp.
go build -buildvcs=false -o bin/$SERVICE ./src/
echo "[$SERVICE] build complete"
