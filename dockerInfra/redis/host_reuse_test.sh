#!/usr/bin/env bash
# host-reuse 分支离线回归：标记文件 / DOCKER_REDIS_REUSE_HOST / 强制开关解析
set -euo pipefail

ROOT="$(cd "$(dirname "$0")" && pwd)"
TMP="$(mktemp -d)"
trap 'rm -rf "$TMP"' EXIT

pass=0
fail=0
assert_eq() {
  local name="$1" got="$2" want="$3"
  if [[ "$got" == "$want" ]]; then
    echo "PASS  $name"
    pass=$((pass + 1))
  else
    echo "FAIL  $name — got=$got want=$want" >&2
    fail=$((fail + 1))
  fi
}

# 抽出 run.sh / health.sh 中的开关解析逻辑做纯函数测
reuse_forced_off() {
  case "${DOCKER_REDIS_REUSE_HOST:-}" in
    0|false|FALSE|no|NO) return 0 ;;
    *) return 1 ;;
  esac
}
reuse_forced_on() {
  case "${DOCKER_REDIS_REUSE_HOST:-}" in
    1|true|TRUE|yes|YES) return 0 ;;
    *) return 1 ;;
  esac
}

MARKER="$TMP/.reuse_host_redis"
enable_host_reuse() {
  printf '%s\n' "$1" >"$MARKER"
}
clear_host_reuse() {
  rm -f "$MARKER"
}

# --- forced off/on ---
DOCKER_REDIS_REUSE_HOST=0
if reuse_forced_off; then echo "PASS  DOCKER_REDIS_REUSE_HOST=0 → forced off"; pass=$((pass+1)); else echo "FAIL forced off"; fail=$((fail+1)); fi
if reuse_forced_on; then echo "FAIL should not be forced on"; fail=$((fail+1)); else echo "PASS  not forced on when 0"; pass=$((pass+1)); fi

DOCKER_REDIS_REUSE_HOST=1
if reuse_forced_on; then echo "PASS  DOCKER_REDIS_REUSE_HOST=1 → forced on"; pass=$((pass+1)); else echo "FAIL forced on"; fail=$((fail+1)); fi
if reuse_forced_off; then echo "FAIL should not be forced off"; fail=$((fail+1)); else echo "PASS  not forced off when 1"; pass=$((pass+1)); fi

unset DOCKER_REDIS_REUSE_HOST
if reuse_forced_off || reuse_forced_on; then
  echo "FAIL unset should be neither forced" >&2
  fail=$((fail + 1))
else
  echo "PASS  unset → neither forced"
  pass=$((pass + 1))
fi

# --- marker lifecycle ---
clear_host_reuse
[[ ! -f "$MARKER" ]]
enable_host_reuse "host port already in use"
assert_eq "marker created" "$(cat "$MARKER")" "host port already in use"
clear_host_reuse
if [[ ! -f "$MARKER" ]]; then
  echo "PASS  marker cleared"
  pass=$((pass + 1))
else
  echo "FAIL  marker still present" >&2
  fail=$((fail + 1))
fi

# --- health.sh 源码含 reuse 分支（防回归删掉）---
if grep -q 'DOCKER_REDIS_REUSE_HOST' "$ROOT/health.sh" && grep -q 'REUSE_MARKER\|.reuse_host_redis' "$ROOT/health.sh"; then
  echo "PASS  health.sh still documents host-reuse"
  pass=$((pass + 1))
else
  echo "FAIL  health.sh missing host-reuse markers" >&2
  fail=$((fail + 1))
fi
if grep -q 'enable_host_reuse\|try_host_reuse' "$ROOT/run.sh"; then
  echo "PASS  run.sh still has host-reuse helpers"
  pass=$((pass + 1))
else
  echo "FAIL  run.sh missing host-reuse helpers" >&2
  fail=$((fail + 1))
fi

echo "=== redis host-reuse tests: pass=$pass fail=$fail ==="
[[ "$fail" -eq 0 ]]
