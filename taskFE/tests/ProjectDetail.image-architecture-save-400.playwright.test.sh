#!/usr/bin/env bash
# Run ProjectDetail image-architecture-save-400 E2E test (跨架构保存镜像 400)
# Usage: ./ProjectDetail.image-architecture-save-400.playwright.test.sh [--headed]
# Note: Uses bundled Chromium via executablePath override for environments without system Chrome.
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
PROJECT_DIR="$(cd "$SCRIPT_DIR/.." && pwd)"

cd "$PROJECT_DIR"

export CI="${CI:-1}"

npx playwright test \
  --config="$SCRIPT_DIR/ProjectDetail.image-architecture-save-400.playwright.config.js" \
  "$SCRIPT_DIR/ProjectDetail.image-architecture-save-400.playwright.test.js" \
  "$@"
