#!/usr/bin/env bash
# OPT-20260722-036：listening/http_code 远端 000 误判离线回归（可进 CI，不依赖本机服务栈）
set -euo pipefail

# 与 scripts/smoke/internal-apis-live.sh 中 listening()/http_code 归一化语义一致
normalize_http_code() {
  local code="${1-}"
  if [[ -z "$code" ]]; then
    code="000"
  fi
  printf '%s' "$code"
}

listening_remote_code() {
  local code
  code=$(normalize_http_code "${1-}")
  # 回归：勿把 curl 已写的 000 再 || echo 000 拼成 000000
  [[ "$code" != "000" && "$code" != "000000" ]]
}

pass=0
fail=0

assert_true() {
  local name="$1"
  shift
  if "$@"; then
    echo "PASS  $name"
    pass=$((pass + 1))
  else
    echo "FAIL  $name" >&2
    fail=$((fail + 1))
  fi
}

assert_false() {
  local name="$1"
  shift
  if "$@"; then
    echo "FAIL  $name" >&2
    fail=$((fail + 1))
  else
    echo "PASS  $name"
    pass=$((pass + 1))
  fi
}

assert_false "empty code → not listening" listening_remote_code ""
assert_false "000 → not listening" listening_remote_code "000"
assert_false "000000 误拼也不算在听" listening_remote_code "000000"
assert_true "502 算在听（边缘有响应）" listening_remote_code "502"
assert_true "200 算在听" listening_remote_code "200"
if [[ "$(normalize_http_code "")" == "000" ]]; then
  echo "PASS  normalize empty → 000"
  pass=$((pass + 1))
else
  echo "FAIL  normalize empty → 000" >&2
  fail=$((fail + 1))
fi
if [[ "$(normalize_http_code 000)" == "000" ]]; then
  echo "PASS  normalize 000 stays 000"
  pass=$((pass + 1))
else
  echo "FAIL  normalize 000 stays 000" >&2
  fail=$((fail + 1))
fi

# 可选：若仓库内仍有 live smoke 脚本，跑 required+不可达主机集成断言
ROOT="$(cd "$(dirname "$0")/../../.." && pwd)"
SMOKE="$ROOT/scripts/smoke/internal-apis-live.sh"
if [[ -x "$SMOKE" ]]; then
  TMPDIR_TEST="$(mktemp -d)"
  trap 'rm -rf "$TMPDIR_TEST"' EXIT
  OUT="$TMPDIR_TEST/smoke.out"
  set +e
  INTERNAL_API_SMOKE_HOST=203.0.113.1 INTERNAL_API_SMOKE_MODE=required \
    bash "$SMOKE" >"$OUT" 2>&1
  rc=$?
  set -e
  if [[ "$rc" -ne 0 ]]; then
    echo "PASS  required+unreachable host exits non-zero (rc=$rc)"
    pass=$((pass + 1))
  else
    echo "FAIL  required+unreachable host should exit non-zero" >&2
    fail=$((fail + 1))
  fi
  if grep -qE 'FAIL  saas-backend:8001|required but not listening' "$OUT"; then
    echo "PASS  required mode records saas not listening as FAIL"
    pass=$((pass + 1))
  else
    echo "FAIL  missing saas not-listening FAIL line" >&2
    fail=$((fail + 1))
  fi
  if grep -qE 'fail=[1-9]' "$OUT"; then
    echo "PASS  summary reports fail>=1"
    pass=$((pass + 1))
  else
    echo "FAIL  summary should report fail>=1" >&2
    fail=$((fail + 1))
  fi
else
  echo "SKIP  live smoke script not present (root scripts/ gitignored) — unit asserts only"
fi

echo "=== listening offline tests: pass=$pass fail=$fail ==="
[[ "$fail" -eq 0 ]]
