#!/usr/bin/env bash
# Run CreateProject Git-repo-top E2E
# Usage: ./CreateProject.git-repo-top.playwright.test.sh [--headed]
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
PROJECT_DIR="$(cd "$SCRIPT_DIR/../.." && pwd)"

cd "$PROJECT_DIR"

if [ "${1:-}" = "--headed" ]; then
  export CI=""
  echo "Running in headed mode..."
else
  export CI="1"
  echo "Running in headless mode..."
fi

npx playwright test \
  --config="$SCRIPT_DIR/CreateProject.git-repo-top.playwright.config.js" \
  "$SCRIPT_DIR/CreateProject.git-repo-top.playwright.test.js" \
  "$@"
