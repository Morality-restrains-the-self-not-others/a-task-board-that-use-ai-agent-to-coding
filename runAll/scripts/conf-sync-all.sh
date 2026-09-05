#!/usr/bin/env bash
set -euo pipefail
ROOT="$(cd "$(dirname "$0")/../.." && pwd)"
cd "$ROOT"
# 两级 conf/<area>/<app>/ 与一级 conf/<app>/（如 task-referral）都扫
for d in "$ROOT"/conf/*/*/ "$ROOT"/conf/*/; do
  [[ -d "$d" ]] || continue
  [[ -f "${d}sync.manifest.yaml" ]] || continue
  name="${d#"$ROOT/conf/"}"
  name="${name%/}"
  python3 "$ROOT/runAll/scripts/conf-sync.py" "$name"
done
