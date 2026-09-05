#!/usr/bin/env bash
# Unit tests: 本机 git-service 启动闸门（ADR-0047）。
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
# shellcheck source=local_gitlab_start_gate.sh
source "$SCRIPT_DIR/local_gitlab_start_gate.sh"

PASS=0
FAIL=0
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

rc=0
local_gitlab_start_allowed "git-service" "false" || rc=$?
assert_rc "default git-service disabled" "$rc" 1

rc=0
local_gitlab_start_allowed "git-service" "true" || rc=$?
assert_rc "git-service enabled" "$rc" 0

rc=0
local_gitlab_start_allowed "git-service" "" || rc=$?
assert_rc "git-service empty treated disabled" "$rc" 1

rc=0
local_gitlab_start_allowed "git-service-tencent-sh-1" "false" || rc=$?
assert_rc "shanghai instance not gated" "$rc" 0

rc=0
local_gitlab_start_allowed "git-service-tencent-sh-1" "true" || rc=$?
assert_rc "shanghai instance still allowed" "$rc" 0

msg="$(print_local_gitlab_start_refused)"
if [[ "$msg" == *"runAllStartEnabled"* && "$msg" == *"conf-local/infra/git-service/config.yaml"* ]]; then
  echo "PASS refuse message names conf key"
  PASS=$((PASS + 1))
else
  echo "FAIL refuse message missing conf key: $msg" >&2
  FAIL=$((FAIL + 1))
fi

echo "ok ($PASS passed, $FAIL failed)"
if [[ "$FAIL" -ne 0 ]]; then
  exit 1
fi
