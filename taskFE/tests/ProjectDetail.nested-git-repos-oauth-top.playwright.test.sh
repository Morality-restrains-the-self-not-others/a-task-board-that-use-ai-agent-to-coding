#!/usr/bin/env bash
set -euo pipefail
ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT"
npx playwright test --config=playwright.config.headless.js \
  tests/ProjectDetail.nested-git-repos-oauth-top.playwright.test.js "$@"
