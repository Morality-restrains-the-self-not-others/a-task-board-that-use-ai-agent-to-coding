#!/usr/bin/env bash
# detect_corrupt_data.sh 离线回归：损坏空壳 / 空目录 / 正常目录 / 不存在 / 缺 mysql 系统库
set -euo pipefail

ROOT="$(cd "$(dirname "$0")" && pwd)"
TMP="$(mktemp -d)"
trap 'rm -rf "$TMP"' EXIT

pass=0
fail=0
assert_detect() {
  local name="$1" path="$2" want="$3"
  local got=0
  if "$ROOT/detect_corrupt_data.sh" "$path"; then
    got=0   # exit 0 = 损坏
  else
    got=1   # exit 1 = 正常/空/不存在
  fi
  if [[ "$got" == "$want" ]]; then
    echo "PASS  $name"
    pass=$((pass + 1))
  else
    echo "FAIL  $name — got=损坏($got) want=$want" >&2
    fail=$((fail + 1))
  fi
}

# 1) 完全空目录（首次初始化）→ 非损坏
EMPTY="$TMP/empty"; mkdir -p "$EMPTY"
assert_detect "完全空目录 → 正常" "$EMPTY" 1

# 2) 非空但缺 ibdata1 → 损坏空壳（本次线上故障形态）
SHELL="$TMP/shell"; mkdir -p "$SHELL"; touch "$SHELL/auto.cnf" "$SHELL/ib_buffer_pool"
assert_detect "缺 ibdata1 空壳 → 损坏" "$SHELL" 0

# 3) 含 ibdata1 + mysql/ 系统库 → 正常
OK="$TMP/ok"; mkdir -p "$OK/mysql" "$OK/performance_schema"; touch "$OK/ibdata1"
assert_detect "含 ibdata1+mysql → 正常" "$OK" 1

# 4) 缺 mysql/ 但含 ibdata1 → 损坏
NO_MYSQL="$TMP/no_mysql"; mkdir -p "$NO_MYSQL/performance_schema"; touch "$NO_MYSQL/ibdata1"
assert_detect "缺 mysql 系统库 → 损坏" "$NO_MYSQL" 0

# 5) 目录不存在 → 非损坏
assert_detect "目录不存在 → 正常" "$TMP/not_exist" 1

echo "----"
echo "detect_corrupt_data tests: $pass passed, $fail failed"
[[ "$fail" -eq 0 ]] || exit 1
