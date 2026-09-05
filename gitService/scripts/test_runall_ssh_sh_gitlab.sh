#!/usr/bin/env bash
# Unit tests for runall_ssh_sh_gitlab.sh (9999 remote lifecycle on Host sh).
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PASS=0
FAIL=0

assert_contains() {
  local name="$1" hay="$2" needle="$3"
  if [[ "$hay" == *"$needle"* ]]; then
    echo "PASS $name"
    PASS=$((PASS + 1))
  else
    echo "FAIL $name: missing [$needle] in [$hay]" >&2
    FAIL=$((FAIL + 1))
  fi
}

assert_not_contains() {
  local name="$1" hay="$2" needle="$3"
  if [[ "$hay" != *"$needle"* ]]; then
    echo "PASS $name"
    PASS=$((PASS + 1))
  else
    echo "FAIL $name: unexpected [$needle] in [$hay]" >&2
    FAIL=$((FAIL + 1))
  fi
}

assert_rc() {
  local name="$1" rc="$2" want="$3"
  if [[ "$rc" -eq "$want" ]]; then
    echo "PASS $name"
    PASS=$((PASS + 1))
  else
    echo "FAIL $name: rc=$rc want=$want" >&2
    FAIL=$((FAIL + 1))
  fi
}

normalize_dry() {
  # bash printf %q escapes spaces as '\ '
  printf '%s' "${1//\\ / }"
}

run_dry() {
  GITSERVICE_SH_SSH_DRY_RUN=1 GITSERVICE_SH_SSH_BIN=echo \
    GITSERVICE_SH_SSH_HOST="${GITSERVICE_SH_SSH_HOST:-sh}" \
    GITSERVICE_SH_COMPOSE_DIR="${GITSERVICE_SH_COMPOSE_DIR:-/opt/daydaymoney/gitservice-tencent-sh-1}" \
    bash "$SCRIPT_DIR/runall_ssh_sh_gitlab.sh" "$1"
}

start_out="$(normalize_dry "$(run_dry start)")"
assert_contains "start-ssh-host" "$start_out" " sh "
assert_contains "start-compose-dir" "$start_out" "/opt/daydaymoney/gitservice-tencent-sh-1"
assert_contains "start-compose-up" "$start_out" "docker compose up -d"
assert_not_contains "start-not-local-deploy" "$start_out" "deploy_tencent_sh_1"
assert_not_contains "start-not-infra-runsh" "$start_out" "gitService/run.sh"

stop_out="$(normalize_dry "$(run_dry stop)")"
assert_contains "stop-ssh-host" "$stop_out" " sh "
assert_contains "stop-compose-stop" "$stop_out" "docker compose stop"
assert_not_contains "stop-not-down" "$stop_out" "compose down"

set +e
bash "$SCRIPT_DIR/runall_ssh_sh_gitlab.sh" >/dev/null 2>&1
rc=$?
set -e
assert_rc "missing-action-fails" "$rc" 2

set +e
bash "$SCRIPT_DIR/runall_ssh_sh_gitlab.sh" restart >/dev/null 2>&1
rc=$?
set -e
assert_rc "unknown-action-fails" "$rc" 2

echo "result pass=$PASS fail=$FAIL"
if [[ "$FAIL" -ne 0 ]]; then
  exit 1
fi
