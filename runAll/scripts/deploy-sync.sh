#!/usr/bin/env bash
# Deploy-host: download pinned artifacts then exec last-good (ADR-0052 / ADR-0027).
set -euo pipefail
RUNALL_ROOT="$(cd "$(dirname "$0")/.." && pwd)"
CUTOVER="${CUTOVER_ENV:-}"
if [[ -z "$CUTOVER" && -n "${DEPLOY_ROOT:-}" && -f "$DEPLOY_ROOT/cutover.env" ]]; then
  CUTOVER="$DEPLOY_ROOT/cutover.env"
fi
if [[ -z "$CUTOVER" && -f "$RUNALL_ROOT/../cutover.env" ]]; then
  CUTOVER="$RUNALL_ROOT/../cutover.env"
fi
if [[ -n "$CUTOVER" && -f "$CUTOVER" ]]; then
  set -a
  # shellcheck disable=SC1091
  source "$CUTOVER"
  set +a
fi
export DEPLOY_MODE="${DEPLOY_MODE:-1}"
CONFIG="${1:-}"
if [[ -z "$CONFIG" ]]; then
  if [[ -n "${CONF_ROOT:-}" ]]; then
    CONFIG="$CONF_ROOT/runAll.yaml"
  elif [[ -n "${DEPLOY_ROOT:-}" ]]; then
    CONFIG="$DEPLOY_ROOT/conf/runAll.yaml"
  else
    CONFIG="$RUNALL_ROOT/../conf/runAll.yaml"
  fi
fi
BIN="$RUNALL_ROOT/bin/runAll"
if [[ ! -x "$BIN" ]]; then
  echo "deploy-sync.sh: missing $BIN — build runAll on a source worktree first" >&2
  exit 1
fi
exec "$BIN" -command deploy-sync -config "$CONFIG"
