#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "${SCRIPT_DIR}/../.." && pwd)"

cd "${PROJECT_ROOT}/../../taskFE"
npx playwright test "tests/Login.phone-password-e164-after-register.playwright.test.js" --config="playwright.verify.config.js"
