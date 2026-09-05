#!/usr/bin/env bash
# Apply dataMigrate SQL files for ai_provider database.
# Uses the shared apply_datamigrate.sh script.
# Called by runAll /api/dev/init-databases (port 9999).
set -euo pipefail
ROOT="$(cd "$(dirname "$0")/../.." && pwd)"
exec bash "$ROOT/db/scripts/apply_datamigrate.sh" "ai_provider" "$ROOT/dataMigrate/taskAiProvider"
