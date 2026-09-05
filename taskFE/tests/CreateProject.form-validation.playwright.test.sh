#!/usr/bin/env bash
# Run CreateProject E2E tests
# Usage: ./CreateProject.form-validation.playwright.test.sh [--headed]
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
PROJECT_DIR="$(cd "$SCRIPT_DIR/../.." && pwd)"
FRONT_DIR="$(cd "$PROJECT_DIR/./app" && pwd)"

cd "$PROJECT_DIR"

if [ "${1:-}" = "--headed" ]; then
  export CI=""
  echo "Running in headed mode..."
else
  export CI="1"
  echo "Running in headless mode..."
fi

npx playwright test \
  --config="$SCRIPT_DIR/../playwright.config.local.js" \
  "$SCRIPT_DIR/CreateProject.form-validation.playwright.test.js" \
  "$@"
