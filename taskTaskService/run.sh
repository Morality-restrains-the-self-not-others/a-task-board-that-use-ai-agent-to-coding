#!/bin/bash
set -euo pipefail
REPO_ROOT="$(cd "$(dirname "$0")/.." && pwd)"
SERVICE="taskTaskService"
cd "$REPO_ROOT/$SERVICE"
echo "[$SERVICE] starting on :8017..."
exec ./bin/$SERVICE
