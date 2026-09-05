#!/usr/bin/env bash
# handle_corrupt_datadir.sh：空壳默认非 0；MYSQL_ALLOW_EMPTY_REINIT=1 才重建。
set -euo pipefail

ROOT="$(cd "$(dirname "$0")" && pwd)"
TMP="$(mktemp -d)"
trap 'rm -rf "$TMP"' EXIT

pass=0
fail=0
assert_exit() {
  local name="$1" want="$2"
  shift 2
  local got=0
  if "$@"; then
    got=0
  else
    got=$?
  fi
  if [[ "$got" == "$want" ]]; then
    echo "PASS  $name"
    pass=$((pass + 1))
  else
    echo "FAIL  $name — got=$got want=$want" >&2
    fail=$((fail + 1))
  fi
}

HANDLE="$ROOT/handle_corrupt_datadir.sh"

# 1) 完全空目录 → 允许启动
EMPTY="$TMP/empty"; mkdir -p "$EMPTY"
assert_exit "空目录 → 0" 0 env -u MYSQL_ALLOW_EMPTY_REINIT "$HANDLE" "$EMPTY"

# 2) 空壳无允许标志 → 拒绝，且目录仍在
SHELL="$TMP/shell"; mkdir -p "$SHELL"; touch "$SHELL/auto.cnf"
assert_exit "空壳默认拒绝" 1 env -u MYSQL_ALLOW_EMPTY_REINIT "$HANDLE" "$SHELL"
if [[ -f "$SHELL/auto.cnf" ]]; then
  echo "PASS  拒绝后空壳仍在"
  pass=$((pass + 1))
else
  echo "FAIL  拒绝后空壳被挪走" >&2
  fail=$((fail + 1))
fi

# 3) 空壳 + MYSQL_ALLOW_EMPTY_REINIT=1 → 0，原目录变成 .corrupt.* 或空目录
ALLOW="$TMP/allow"; mkdir -p "$ALLOW"; touch "$ALLOW/auto.cnf"
assert_exit "空壳显式重建" 0 env MYSQL_ALLOW_EMPTY_REINIT=1 "$HANDLE" "$ALLOW"
if [[ -d "$ALLOW" ]] && [[ -z "$(ls -A "$ALLOW")" ]]; then
  echo "PASS  重建后为空目录"
  pass=$((pass + 1))
else
  echo "FAIL  重建后不是空目录" >&2
  fail=$((fail + 1))
fi
if compgen -G "$TMP/allow.corrupt.*" >/dev/null; then
  echo "PASS  原空壳已 mv 为 .corrupt.*"
  pass=$((pass + 1))
else
  echo "FAIL  未留下 .corrupt.* 备份" >&2
  fail=$((fail + 1))
fi

# 4) 指向空壳的符号链接 + 拒绝：链接与目标都还在
TGT="$TMP/linktgt"; mkdir -p "$TGT"; touch "$TGT/auto.cnf"
LINK="$TMP/linkdata"; ln -s "$TGT" "$LINK"
assert_exit "符号链接空壳拒绝" 1 env -u MYSQL_ALLOW_EMPTY_REINIT "$HANDLE" "$LINK"
if [[ -L "$LINK" ]] && [[ -f "$TGT/auto.cnf" ]]; then
  echo "PASS  拒绝后未拆链接、未动目标"
  pass=$((pass + 1))
else
  echo "FAIL  拒绝路径改动了符号链接或目标" >&2
  fail=$((fail + 1))
fi

# 5) 符号链接 + ALLOW：只拆链接，目标原文件仍在
assert_exit "符号链接空壳显式重建" 0 env MYSQL_ALLOW_EMPTY_REINIT=1 "$HANDLE" "$LINK"
if [[ -d "$LINK" && ! -L "$LINK" ]] && [[ -f "$TGT/auto.cnf" ]]; then
  echo "PASS  重建只拆链接、不 mv 目标"
  pass=$((pass + 1))
else
  echo "FAIL  重建动到了符号链接目标" >&2
  fail=$((fail + 1))
fi

# 6) 正常 ibdata1+mysql → 0
OK="$TMP/ok"; mkdir -p "$OK/mysql"; touch "$OK/ibdata1"
assert_exit "正常 datadir → 0" 0 env -u MYSQL_ALLOW_EMPTY_REINIT "$HANDLE" "$OK"

echo "----"
echo "handle_corrupt_datadir tests: $pass passed, $fail failed"
[[ "$fail" -eq 0 ]] || exit 1
