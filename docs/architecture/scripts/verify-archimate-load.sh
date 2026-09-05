#!/usr/bin/env bash
# 验证 docs/architecture/*.archimate 可被 Archi CLI 加载。
# 用法（仓库根 /tmp/ram-work 或 docs 仓库内均可）:
#   ./docs/architecture/scripts/verify-archimate-load.sh [model.archimate ...]
# 未传参时校验 docs/architecture/v*-*.archimate
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
ARCH_DIR="$(cd "$SCRIPT_DIR/.." && pwd)"
# monorepo root is parent of docs/ when script lives under docs/architecture/scripts
if [[ -x "$ARCH_DIR/Archi/Archi" ]]; then
  ROOT="$(cd "$ARCH_DIR/../.." && pwd)"
  ARCHI="$ARCH_DIR/Archi/Archi"
  MODEL_GLOB_DIR="$ARCH_DIR"
elif [[ -x "$(pwd)/docs/architecture/Archi/Archi" ]]; then
  ROOT="$(pwd)"
  ARCHI="$ROOT/docs/architecture/Archi/Archi"
  MODEL_GLOB_DIR="$ROOT/docs/architecture"
else
  echo "ERROR: Archi CLI not found" >&2
  exit 1
fi
cd "$ROOT"

models=("$@")
if [[ ${#models[@]} -eq 0 ]]; then
  mapfile -t models < <(find "$MODEL_GLOB_DIR" -maxdepth 1 -name 'v*-*.archimate' | sort)
fi

if [[ ${#models[@]} -eq 0 ]]; then
  echo "ERROR: no .archimate models found" >&2
  exit 1
fi

fail=0
for m in "${models[@]}"; do
  if [[ "$m" != *.archimate ]]; then
    echo "FAIL: path must end with .archimate — got: $m" >&2
    fail=1
    continue
  fi
  if [[ ! -f "$m" ]]; then
    echo "FAIL: missing file: $m" >&2
    fail=1
    continue
  fi
  abs="$(cd "$(dirname "$m")" && pwd)/$(basename "$m")"
  out="$(mktemp)"
  set +e
  xvfb-run -a "$ARCHI" -application com.archimatetool.commandline.app \
    -consoleLog -nosplash --loadModel "$abs" -a >"$out" 2>&1
  rc=$?
  set -e
  if grep -E 'Could not load model|Application error' "$out" >/dev/null 2>&1; then
    echo "FAIL: $m (load error in log, exit=$rc)"
    grep -En 'Could not load model|Application error|Loaded model' "$out" | head -10 || true
    fail=1
  elif ! grep -q 'Loaded model:' "$out"; then
    echo "FAIL: $m (no 'Loaded model:' line, exit=$rc)"
    tail -20 "$out"
    fail=1
  else
    label="$(grep -o "Loaded model: '[^']*'" "$out" | head -1)"
    echo "OK: $m — $label"
  fi
  rm -f "$out"
done

if [[ "$fail" -ne 0 ]]; then
  echo "verify-archimate-load: FAILED" >&2
  exit 1
fi
echo "verify-archimate-load: all OK (${#models[@]} models)"
