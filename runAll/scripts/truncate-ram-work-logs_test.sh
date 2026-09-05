#!/usr/bin/env bash
# Self-check: truncate script must prefer runAll truncate API, never rm logs/*.log,
# and archive a short window before truncation (OPT-20260823-037).
set -euo pipefail
ROOT="$(cd "$(dirname "$0")/../.." && pwd)"
SCRIPT="${ROOT}/runAll/scripts/truncate-ram-work-logs.sh"
LIB="${ROOT}/runAll/scripts/lib/log-archive.sh"

fail() { echo "FAIL: $*" >&2; exit 1; }

[[ -f "$SCRIPT" ]] || fail "missing $SCRIPT"
grep -q '/api/logs/clear-all' "$SCRIPT" || fail "must call /api/logs/clear-all"
grep -q 'RUNALL_UI_PORT' "$SCRIPT" || fail "must honor RUNALL_UI_PORT"
grep -q 'FileServiceLogSink.TruncateAll\|skipped shell truncate' "$SCRIPT" || fail "must skip shell truncate when API ok"
grep -q 'truncate_taskgateway_logs' "$SCRIPT" || fail "must truncate taskGateway via dedicated helper"
grep -q 'docker exec' "$SCRIPT" || fail "must docker-exec truncate APISIX-owned logs"
grep -q 'TASKGATEWAY_APISIX_CONTAINER' "$SCRIPT" || fail "must honor TASKGATEWAY_APISIX_CONTAINER"
grep -q 'chmod a+rw' "$SCRIPT" || fail "must chmod a+rw after container truncate"

# Short-window archive wiring (OPT-20260823-037): source the lib, call archive before truncations, prune after.
[[ -f "$LIB" ]] || fail "missing lib $LIB"
grep -q 'lib/log-archive.sh' "$SCRIPT" || fail "must source lib/log-archive.sh"
grep -q 'TRUNCATE_ARCHIVE_ENABLED' "$SCRIPT" || fail "must honor TRUNCATE_ARCHIVE_ENABLED"
grep -q 'TRUNCATE_ARCHIVE_KEEP' "$SCRIPT" || fail "must honor TRUNCATE_ARCHIVE_KEEP"
[[ "$(grep -c 'archive_short_window' "$SCRIPT")" -ge 4 ]] || fail "must call archive_short_window before each truncation dir"
grep -q 'prune_archive_batches' "$SCRIPT" || fail "must prune old archive batches"
grep -q '^archive_short_window()' "$LIB" || fail "lib must define archive_short_window"
grep -q '^prune_archive_batches()' "$LIB" || fail "lib must define prune_archive_batches"

# OPT-20260905-005: runall-console.log keep-tail on shell fallback truncate
grep -q 'RUNALL_CONSOLE_KEEP_TAIL_BYTES' "$SCRIPT" || fail "must honor RUNALL_CONSOLE_KEEP_TAIL_BYTES"
grep -q 'truncate_keep_tail' "$SCRIPT" || fail "must define truncate_keep_tail for console log"
grep -q 'runall-console.log' "$SCRIPT" || fail "must special-case runall-console.log"

# Forbid rm of tee log globs (temp mktemp cleanup of curl body is ok) — applies to script and lib.
for f in "$SCRIPT" "$LIB"; do
  if grep -nE 'rm[[:space:]]+(-[a-zA-Z]*f?[[:space:]]+)*("?\$\{?RUNALL_LOG_ROOT\}?/?|"?\$\{?ROOT\}?/logs/|\*\.log)' "$f"; then
    fail "$f must not rm tee log files"
  fi
  if grep -nE 'rm[[:space:]]+-rf?[[:space:]]+.*logs' "$f"; then
    fail "$f must not rm -rf logs"
  fi
done

echo "OK: truncate-ram-work-logs.sh prefers /api/logs/clear-all, avoids rm of tee logs, archives short window before truncate"
