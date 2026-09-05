#!/bin/bash
set -euo pipefail
REPO_ROOT="$(cd "$(dirname "$0")/.." && pwd)"
SERVICE="taskProjectService"
cd "$REPO_ROOT/$SERVICE"
echo "[$SERVICE] starting on :8016..."
exec ./bin/$SERVICE
