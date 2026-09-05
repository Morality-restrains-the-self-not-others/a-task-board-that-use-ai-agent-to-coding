#!/usr/bin/env bash
# task_budget schema is handled by dataMigrate/taskBudget/*.sql
# (applied via migrate_script → apply_datamigrate.sh).
# This init script is intentionally a no-op.
set -euo pipefail
echo "[task-budget] init.sh: no-op (schema handled by dataMigrate)"
exit 0
