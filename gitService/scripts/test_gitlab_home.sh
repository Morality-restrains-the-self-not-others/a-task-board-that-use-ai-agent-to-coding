#!/usr/bin/env bash
# Unit tests for gitlab_home.sh (T1–T3).
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
# shellcheck source=gitlab_home.sh
source "$SCRIPT_DIR/gitlab_home.sh"

PASS=0
FAIL=0
assert_eq() {
  local name="$1" got="$2" want="$3"
  if [[ "$got" == "$want" ]]; then
    echo "PASS $name"
    PASS=$((PASS + 1))
  else
    echo "FAIL $name: got=[$got] want=[$want]" >&2
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

TMP="$(mktemp -d)"
trap 'rm -rf "$TMP"' EXIT

# T1: default path
unset GITLAB_HOME
export HOME="$TMP/home"
unset XDG_DATA_HOME
mkdir -p "$HOME"
got="$(resolve_gitlab_home "")"
assert_eq "T1-default" "$got" "$TMP/home/.local/share/daydaymoney/gitService"

export XDG_DATA_HOME="$TMP/xdg"
got="$(resolve_gitlab_home "")"
assert_eq "T1-xdg" "$got" "$TMP/xdg/daydaymoney/gitService"

export GITLAB_HOME="$TMP/custom"
got="$(resolve_gitlab_home "$TMP/from-conf")"
assert_eq "T1-env-wins" "$got" "$TMP/custom"
unset GITLAB_HOME
got="$(resolve_gitlab_home "$TMP/from-conf")"
assert_eq "T1-conf" "$got" "$TMP/from-conf"

# T2: tmpfs reject (simulate via findmnt if path is on tmpfs; else force fstype check)
# Create a volatile marker by testing assert against known tmpfs if available.
VOLATILE_ROOT=""
if findmnt -T /tmp -o FSTYPE -n 2>/dev/null | grep -qx tmpfs; then
  VOLATILE_ROOT="$TMP/on-tmp"
  mkdir -p "$VOLATILE_ROOT"
fi
if [[ -n "$VOLATILE_ROOT" ]]; then
  unset GITLAB_HOME_ALLOW_TMPFS
  set +e
  assert_gitlab_home_durable "$VOLATILE_ROOT" >/dev/null 2>&1
  rc=$?
  set -e
  assert_rc "T2-tmpfs-reject" "$rc" 1
  export GITLAB_HOME_ALLOW_TMPFS=1
  set +e
  assert_gitlab_home_durable "$VOLATILE_ROOT" >/dev/null 2>&1
  rc=$?
  set -e
  assert_rc "T2-tmpfs-allow" "$rc" 0
  unset GITLAB_HOME_ALLOW_TMPFS
else
  echo "SKIP T2 ( /tmp not tmpfs on this host )"
fi

# T3: migrate predicate
LEG="$TMP/legacy"
TGT="$TMP/target"
mkdir -p "$LEG/data/postgresql/data" "$TGT"
: >"$LEG/data/bootstrapped"
set +e
legacy_needs_migrate "$LEG" "$TGT"
rc=$?
set -e
assert_rc "T3-needs-migrate" "$rc" 0

mkdir -p "$TGT/data/postgresql/data"
: >"$TGT/data/bootstrapped"
set +e
legacy_needs_migrate "$LEG" "$TGT"
rc=$?
set -e
assert_rc "T3-skip-when-target-ready" "$rc" 1

echo "--- results: PASS=$PASS FAIL=$FAIL ---"
[[ "$FAIL" -eq 0 ]]
