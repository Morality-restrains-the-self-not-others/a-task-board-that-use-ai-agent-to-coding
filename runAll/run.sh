#!/bin/bash
set -euo pipefail
# App processes must not inherit shell HTTP(S)_PROXY (dev-only network accel).
# See .ai/01_project_constraints/23_app_startup_no_env_proxy.md
unset http_proxy https_proxy HTTP_PROXY HTTPS_PROXY ALL_PROXY all_proxy || true

# ── Infrastructure host (SSoT: conf/base.yaml infraHost) ──────────────────
# Default 10.2.150.68 (production LAN IP). Override in calling shell for dev:
#   INFRA_HOST=127.0.0.1 bash runAll/run.sh
# All Go confload services + runAll orchestrator resolve ${INFRA_HOST:-default}
# from this env var; see shareLib/confload/load.go resolveEnvVars().
export INFRA_HOST="${INFRA_HOST:-10.2.150.68}"

cd "$(dirname "$0")"
ROOT="$(cd .. && pwd)"
# clone-run: ELF live in $ROOT/bin and there is no taskAuth/ source tree.
# cutover.env (written by up.sh) is the deploy-env SSOT — source it, do not re-list keys.
if [[ -x "$ROOT/bin/runAll" && ! -e "$ROOT/taskAuth" ]]; then
	CUTOVER="${CUTOVER_ENV:-$ROOT/cutover.env}"
	if [[ ! -f "$CUTOVER" ]]; then
		echo "ERROR: clone-run requires $CUTOVER (run ./scripts/up.sh first)" >&2
		exit 1
	fi
	echo "==> clone-run layout; sourcing $CUTOVER"
	set -a
	# shellcheck disable=SC1091
	source "$CUTOVER"
	set +a
fi
LOG="${RUNALL_CONSOLE_LOG:-$ROOT/logs/runall-console.log}"
BIN="${RUNALL_BIN:-./bin/runAll}"
CONFIG="${RUNALL_CONFIG:-../conf/runAll.yaml}"

# doctor / takeover / build-all need the caller's tty and exit code.
cli_command=""
prev=""
for arg in "$@"; do
	if [[ "$prev" == "-command" || "$prev" == "--command" ]]; then
		cli_command="$arg"
	fi
	case "$arg" in
	-command=*|--command=*) cli_command="${arg#*=}" ;;
	esac
	prev="$arg"
done

want_foreground=0
if [[ "${RUNALL_FOREGROUND:-0}" == "1" ]]; then
	want_foreground=1
fi
case "$cli_command" in
doctor | takeover | build-all) want_foreground=1 ;;
esac

if [[ "${RUNALL_SKIP_BUILD:-0}" != "1" ]]; then
	echo "==> Building runAll..."
	./build.sh
fi

if [[ ! -x "$BIN" ]]; then
	echo "ERROR: runAll binary not executable: $BIN" >&2
	exit 1
fi

if [[ "$want_foreground" -eq 1 ]]; then
	echo "==> Starting runAll (foreground)..."
	exec "$BIN" --config "$CONFIG" "$@"
fi

# Independent session: do not remain the screen/tty foreground job.
# 2026-08-20 screen -S run ran ./bin/runAll in the foreground (pid=pgid,
# sid=bash); a targeted SIGTERM (detail=terminated) tore the orchestrator down.
# setsid -f always forks so this script is never the session leader; without
# -w the parent exits immediately (do not background with &: $! would be the
# short-lived setsid parent, not runAll).
mkdir -p "$(dirname "$LOG")"
echo "==> Starting runAll (setsid nohup; log $LOG)..."
setsid -f nohup "$BIN" --config "$CONFIG" "$@" >>"$LOG" 2>&1 < /dev/null
echo "==> runAll detached  UI http://127.0.0.1:9999  log=$LOG"
