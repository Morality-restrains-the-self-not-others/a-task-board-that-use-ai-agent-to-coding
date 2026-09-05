#!/usr/bin/env bash
# Source-tree precise compile → gitignored deploy-binaries/ (ADR-0052 / ADR-0027).
# Compiles registered or named runAll services; does not stop/start processes.
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
META_ROOT="${META_ROOT:-$(cd "$SCRIPT_DIR/../.." && pwd)}"
CONF="${PRECISE_COMPILE_CONF:-$META_ROOT/conf/runAll.yaml}"
REG_FILE="${RUNALL_PRECISE_RESTART_FILE:-$META_ROOT/.runall/precise_restart_services.txt}"
DEST="${PRECISE_COMPILE_DEST:-${DEST:-$META_ROOT/deploy-binaries}}"
PY="$SCRIPT_DIR/precise_compile.py"
COLLECT="$SCRIPT_DIR/collect-deploy-binaries.sh"

log() { echo "[precise-compile] $*" >&2; }

ALL=0
INSTALL=0
INSTALL_ROOT="${DEPLOY_ROOT:-$HOME/bin/daydaymoney-deploy}"
NAMES=()
while [[ $# -gt 0 ]]; do
  case "$1" in
    --all) ALL=1; shift ;;
    --install) INSTALL=1; shift ;;
    --install-root) INSTALL_ROOT="$2"; shift 2 ;;
    --dest) DEST="$2"; shift 2 ;;
    --conf) CONF="$2"; shift 2 ;;
    --help|-h)
      echo "usage: precise-compile.sh [--all] [--install] [--install-root DIR] [--dest DIR] [service-name...]" >&2
      echo "default: services in $REG_FILE" >&2
      echo "--install: after collect, install deploy-binaries into \${DEPLOY_ROOT:-$HOME/bin/daydaymoney-deploy} (OPT-20260902-008)" >&2
      exit 0
      ;;
    --) shift; NAMES+=("$@"); break ;;
    -*)
      echo "precise-compile: unknown flag $1" >&2
      exit 2
      ;;
    *) NAMES+=("$1"); shift ;;
  esac
done

RESOLVE=(python3 "$PY" --conf "$CONF")
if [[ "$ALL" == "1" ]]; then
  RESOLVE+=(--all)
elif [[ ${#NAMES[@]} -gt 0 ]]; then
  RESOLVE+=(--names "${NAMES[@]}")
else
  RESOLVE+=(--registry "$REG_FILE")
fi

TSV="$("${RESOLVE[@]}")" || exit $?

compiled=0
if [[ -n "$TSV" ]]; then
  while IFS=$'\t' read -r name wd cmd; do
    [[ -n "$name" ]] || continue
    dir="$META_ROOT"
    if [[ -n "$wd" && "$wd" != "." ]]; then
      dir="$META_ROOT/$wd"
    fi
    if [[ ! -d "$dir" ]]; then
      log "missing working_dir $dir for $name"
      exit 1
    fi
    log "$name: $cmd (cwd=$dir)"
    (
      cd "$dir"
      unset DEPLOY_MODE
      bash -c "$cmd"
    )
    compiled=$((compiled + 1))
  done <<< "$TSV"
fi

if [[ "$compiled" -eq 0 ]]; then
  log "no build_command ran (services skipped or empty resolve)"
fi

log "collect → $DEST"
mkdir -p "$DEST"
COLLECT_SKIP_SHA=1 META_ROOT="$META_ROOT" bash "$COLLECT" "$DEST"
if [[ "$INSTALL" == "1" ]]; then
  mkdir -p "$INSTALL_ROOT"
  log "install $DEST → $INSTALL_ROOT"
  bash "$SCRIPT_DIR/install-local-artifacts.sh" "$DEST" "$INSTALL_ROOT"
fi
log "done ($compiled compiled)"
