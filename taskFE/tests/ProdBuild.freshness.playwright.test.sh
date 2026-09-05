#!/usr/bin/env bash
set -euo pipefail
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
TASKFE_DIR="$(cd "${SCRIPT_DIR}/.." && pwd)"
cd "${TASKFE_DIR}"
# 产物/源码新鲜度校验（OPT-20260824-056）：默认仅告警，REQUIRE_FRESH=1 时漂移即失败
npx playwright test "tests/ProdBuild.freshness.playwright.test.js" --config="playwright.verify.config.js"
