#!/usr/bin/env bash
# Apply dataMigrate SQL files for container database.
# Uses the shared apply_datamigrate.sh script.
# Called by runAll /api/dev/init-databases (port 9999).
set -euo pipefail
ROOT="$(cd "$(dirname "$0")/../.." && pwd)"
exec bash "$ROOT/db/scripts/apply_datamigrate.sh" "container" "$ROOT/dataMigrate/taskCredentialService"
