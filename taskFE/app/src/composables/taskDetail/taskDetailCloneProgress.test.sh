#!/bin/bash
set -euo pipefail
cd "$(dirname "$0")/../../.."
npx vitest run src/composables/taskDetail/taskDetailCloneProgress.test.js
