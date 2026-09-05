#!/usr/bin/env bash
set -euo pipefail
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
APP_DIR="$(cd "${SCRIPT_DIR}/../../.." && pwd)"
cd "${APP_DIR}"
exec npm test -- --run "src/components/task-detail/TaskDetailConversationFeed.execution-details-embed.test.js"
