#!/usr/bin/env bash
# Regression: Django collectstatic must not run (or log) during taskFE build.
# Django saas-backend retired 2026-07-30; Docker nginx serves public/html.
set -euo pipefail
script_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
fail=0

bash -n "$script_dir/runall-lifecycle.sh" || fail=1

pkg="$script_dir/../package.json"
build_line="$(python3 -c "import json; print(json.load(open('$pkg'))['scripts']['build'])")"
if [[ "$build_line" == *collectstatic* ]]; then
  echo "FAIL: package.json build must not call collectstatic (got: $build_line)" >&2
  fail=1
fi
if ! grep -q '"build:vite"' "$pkg"; then
  echo "FAIL: package.json must define build:vite" >&2
  fail=1
fi

# Accidental callers of the retired script must stay silent (no Django log noise).
if [[ -f "$script_dir/collectstatic-after-vite.sh" ]]; then
  bash -n "$script_dir/collectstatic-after-vite.sh" || fail=1
  out="$(bash "$script_dir/collectstatic-after-vite.sh" 2>&1 || true)"
  if [[ "$out" == *[Dd]jango* ]] || [[ "$out" == *collectstatic* ]] || [[ "$out" == *task2app* ]]; then
    echo "FAIL: retired collectstatic-after-vite.sh must be silent; got: $out" >&2
    fail=1
  fi
  if ! bash "$script_dir/collectstatic-after-vite.sh"; then
    echo "FAIL: retired collectstatic-after-vite.sh must exit 0" >&2
    fail=1
  fi
fi

# runall-lifecycle build must not invoke collectstatic by name
if grep -qE 'collectstatic' "$script_dir/runall-lifecycle.sh"; then
  echo "FAIL: runall-lifecycle.sh must not reference collectstatic" >&2
  fail=1
fi

if [[ "$fail" -ne 0 ]]; then
  echo "test_collectstatic_after_vite.sh FAILED" >&2
  exit 1
fi
echo "test_collectstatic_after_vite.sh OK"
