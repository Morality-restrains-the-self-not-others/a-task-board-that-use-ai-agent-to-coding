#!/usr/bin/env bash
# Run CreateProject OAuth button E2E tests
# Usage: ./CreateProject.oauth-button.playwright.test.sh [--headed]
# Note: Uses bundled Chromium via executablePath override for environments without system Chrome.
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
PROJECT_DIR="$(cd "$SCRIPT_DIR/../.." && pwd)"

cd "$PROJECT_DIR"

export CI="${CI:-1}"

# Use a local config that doesn't require system Chrome (channel:'chrome')
npx playwright test \
  --config="$SCRIPT_DIR/CreateProject.oauth-button.playwright.config.js" \
  "$SCRIPT_DIR/CreateProject.oauth-button.playwright.test.js" \
  "$@"
