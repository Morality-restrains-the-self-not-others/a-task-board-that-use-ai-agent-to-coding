#!/usr/bin/env bash
set -euo pipefail
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
TASKFE_DIR="$(cd "${SCRIPT_DIR}/.." && pwd)"
cd "${TASKFE_DIR}"
npx playwright test "tests/Navbar.git-service-empty-to-pricing.playwright.test.js" --config="playwright.verify.config.js"
