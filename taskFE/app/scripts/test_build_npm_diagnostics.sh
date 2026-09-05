#!/usr/bin/env bash
# 验证：taskFE 原子构建在 node_modules 缺失时须给出可操作诊断（OPT-20260810-010），
# 并保留 npm ci 失败时的 exit 127。禁止对真实 app/node_modules 做破坏性仿真。
set -euo pipefail
script_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
fail=0

bash -n "$script_dir/runall-lifecycle.sh" || fail=1
bash -n "$script_dir/atomic-vite-build.sh" || fail=1

atomic="$(cat "$script_dir/atomic-vite-build.sh")"

if ! printf '%s\n' "$atomic" | grep -qE 'node_modules/.bin/vite'; then
  echo "FAIL: atomic-vite-build.sh 缺少 node_modules/.bin/vite 就绪检查" >&2
  fail=1
fi
check_line="$(printf '%s\n' "$atomic" | grep -nE 'node_modules/.bin/vite' | head -1 | cut -d: -f1 || true)"
build_line="$(printf '%s\n' "$atomic" | grep -nE 'vite build' | head -1 | cut -d: -f1 || true)"
if [[ -z "$check_line" || -z "$build_line" || "$check_line" -ge "$build_line" ]]; then
  echo "FAIL: node_modules 检查必须位于 vite build 之前 (check=$check_line, build=$build_line)" >&2
  fail=1
fi

if ! grep -qE 'atomic-vite-build\.sh' "$script_dir/runall-lifecycle.sh"; then
  echo "FAIL: runall-lifecycle.sh build 须调用 atomic-vite-build.sh" >&2
  fail=1
fi

if ! printf '%s\n' "$atomic" | grep -qE 'vite 缺失：自动执行 npm ci|node_modules/.bin/vite 缺失'; then
  echo "FAIL: 诊断文案应提示 vite 缺失并自动 npm ci" >&2
  fail=1
fi
if ! printf '%s\n' "$atomic" | grep -qE 'exit 127'; then
  echo "FAIL: 诊断应保留原 exit 127 语义" >&2
  fail=1
fi

if [[ "$fail" -ne 0 ]]; then
  echo "test_build_npm_diagnostics.sh FAILED" >&2
  exit 1
fi
echo "test_build_npm_diagnostics.sh OK"
