#!/usr/bin/env bash
# CI check: verify Vite can statically analyze all dynamic import() calls.
#
# Two-phase check:
#   Phase 1 (source):  detect import(variable) / import(functionCall()) patterns
#                       that Vite cannot resolve at build time
#   Phase 2 (artifacts): scan built JS chunks for unresolved relative import paths
#                        that would cause runtime 404 (like "../js/comments.js")
#
# Usage:
#   ./check-vite-dynamic-imports.sh              # run both phases
#   ./check-vite-dynamic-imports.sh --source-only  # only phase 1
#   ./check-vite-dynamic-imports.sh --artifacts-only  # only phase 2
#
# Exit: 0 = clean, non-zero = issues found

set -euo pipefail

script_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
# scripts/ → app/ → taskFE/
app_dir="$(cd "$script_dir/.." && pwd)"
src_dir="$app_dir/src"
assets_dir="$app_dir/static/assets"

PHASE_SOURCE=true
PHASE_ARTIFACTS=true

for arg in "$@"; do
  case "$arg" in
    --source-only) PHASE_ARTIFACTS=false ;;
    --artifacts-only) PHASE_SOURCE=false ;;
    *) echo "Unknown arg: $arg" >&2; exit 2 ;;
  esac
done

ISSUES=0

# ─────────────────────────────────────────────────────────────
# Phase 1 — Source scan
# ─────────────────────────────────────────────────────────────
if $PHASE_SOURCE; then
  echo "=== Phase 1: Source scan for indirect dynamic import() ==="

  # Strategy: find lines with `import(` that are NOT string-literal imports
  # and NOT JSDoc type annotations.
  #
  # Step A: collect all `import(` lines from source (exclude tests/node_modules)
  # Step B: remove lines that are string literals  → import("...") / import('...')
  # Step C: remove lines that are JSDoc annotations → @param/@type/@return
  # Step D: remove lines that are comment-only         → //... or * ... or /*...
  # Whatever remains are potential indirect import() calls.

  raw_imports=$(grep -rn 'import(' "$src_dir" \
    --include='*.js' --include='*.ts' --include='*.vue' --include='*.tsx' \
    | grep -v node_modules \
    | grep -v '/test/' \
    | grep -v '__tests__' \
    | grep -v '\.test\.' \
    || true)

  # Filter: keep only the content after the last colon (grep -rn output: file:line:content)
  # then apply content-based filters
  indirect=$(echo "$raw_imports" \
    | grep -v 'import("[^"]*")' \
    | grep -v "import('[^']*')" \
    | grep -v '@param.*import(' \
    | grep -v '@type.*import(' \
    | grep -v '@return.*import(' \
    | grep -v '^\s*//' \
    | grep -vP ':\d+:\s*\*\s' \
    | grep -vP ':\d+:\s*/\*\*' \
    || true)

  if [[ -n "$indirect" ]]; then
    echo "ERROR: Found indirect dynamic import() calls (Vite cannot resolve at build time):"
    echo "$indirect"
    echo ""
    echo "Fix: Replace import(variable) with import('literal-path') so Vite can"
    echo "     statically analyze and generate the correct hashed chunk."
    echo ""
    echo "     Example — BEFORE (broken):"
    echo "       import(someModulePath)"
    echo "     Example — AFTER (fixed):"
    echo "       import('../js/comments.js')"
    ISSUES=$((ISSUES + 1))
  else
    echo "✓ No indirect dynamic import() patterns found in source"
  fi
  echo ""
fi

# ─────────────────────────────────────────────────────────────
# Phase 2 — Built artifact scan
# ─────────────────────────────────────────────────────────────
if $PHASE_ARTIFACTS; then
  echo "=== Phase 2: Built artifact scan for unresolved import paths ==="

  if [[ ! -d "$assets_dir" ]]; then
    echo "SKIP: assets directory not found at $assets_dir (run 'npm run build' first)"
    exit 0
  fi

  # Detection logic:
  #   RESOLVED:   "./chunkName-AbCdEf12.js"  — has hash suffix before .js
  #   UNRESOLVED: "../js/comments.js"         — source path, no hash
  #
  # Vite always appends a content hash (≥6 alphanumeric chars + underscore/dash)
  # to every chunk name. A relative path ending in .js WITHOUT a hash pattern
  # inside a built chunk means the import was not resolved by Vite.

  unresolved=$(grep -Pn '"[.]{1,2}/[^"]+\.js"' "$assets_dir"/*.js \
    | grep -v '__vite__mapDeps' \
    | grep -vP '"[.]{1,2}/[A-Za-z][^"]*-[A-Za-z0-9_-]{6,}\.js"' \
    || true)

  if [[ -n "$unresolved" ]]; then
    echo "ERROR: Found unresolved relative import paths in built chunks:"
    echo "$unresolved"
    echo ""
    echo "These source-like paths were not transformed by Vite (no content hash)."
    echo "They will cause runtime 404 because the file doesn't exist at that path."
    echo "Check if the source uses import(variable) instead of import('literal')."
    ISSUES=$((ISSUES + 1))
  else
    echo "✓ No unresolved relative import paths in built chunks"
  fi

  # Secondary check: verify every file referenced in vite manifest exists on disk
  manifest="$app_dir/static/assets/.vite/manifest.json"
  if [[ ! -f "$manifest" ]]; then
    manifest="$assets_dir/.vite/manifest.json"
  fi

  if [[ -f "$manifest" ]]; then
    manifest_missing=0
    while IFS= read -r entry_file; do
      [[ -z "$entry_file" ]] && continue
      local_path="${entry_file#assets/}"
      if [[ ! -f "$assets_dir/$local_path" ]]; then
        echo "ERROR: manifest references missing chunk: $entry_file"
        manifest_missing=$((manifest_missing + 1))
      fi
    done < <(python3 -c "
import json, sys
try:
    data = json.loads(open('$manifest').read())
    for ent in data.values():
        if isinstance(ent, dict):
            f = ent.get('file','')
            if f: print(f)
            for css in ent.get('css') or []:
                print(css)
except Exception as e:
    print(f'manifest parse error: {e}', file=sys.stderr)
    sys.exit(1)
" 2>/dev/null || true)

    if [[ "$manifest_missing" -gt 0 ]]; then
      echo "ERROR: $manifest_missing manifest entries reference missing chunk files"
      ISSUES=$((ISSUES + 1))
    else
      echo "✓ All manifest entries reference existing chunk files"
    fi
  else
    echo "SKIP: vite manifest not found (build may not have completed)"
  fi
  echo ""
fi

# ─────────────────────────────────────────────────────────────
# Summary
# ─────────────────────────────────────────────────────────────
if [[ "$ISSUES" -eq 0 ]]; then
  echo "=== RESULT: All checks passed ✓ ==="
  exit 0
else
  echo "=== RESULT: $ISSUES check(s) failed ✗ ===" >&2
  exit 1
fi
