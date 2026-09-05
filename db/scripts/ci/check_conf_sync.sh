#!/usr/bin/env bash
# Ensure conf/ matches conf-sync-all.sh output (entire tree, D4=B).
# Runs the sync against a temp copy of conf/ so the real working tree stays
# clean (OPT-20260901-024). Exit 1 on drift, 2 if preconditions missing.
set -euo pipefail
ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../../.." && pwd)"
cd "$ROOT"
# OPT-20260806-057: django 目录退役，锚点切到 conf/base.yaml
if ! [[ -f conf/base.yaml ]]; then
  echo "check_conf_sync: missing conf/base.yaml" >&2
  exit 2
fi

TMP="$(mktemp -d)"
trap 'rm -rf "$TMP"' EXIT

# Copy conf/ and the sync scripts so conf-sync resolves ROOT to $TMP
# (repo_root() honors CONF_ROOT / __file__ walk) and never writes the real conf/.
mkdir -p "$TMP/runAll/scripts"
cp -a conf "$TMP/conf"
for f in conf-sync-all.sh conf-sync.py conf_lib.py conf_local.py; do
  cp "$ROOT/runAll/scripts/$f" "$TMP/runAll/scripts/"
done
chmod +x "$TMP/runAll/scripts/conf-sync-all.sh"

if ! CONF_ROOT="$TMP/conf" bash "$TMP/runAll/scripts/conf-sync-all.sh"; then
  echo "check_conf_sync: conf-sync-all.sh failed in temp copy" >&2
  exit 1
fi

python3 - "$ROOT" "$TMP" <<'PY'
import sys
from pathlib import Path

root = Path(sys.argv[1])
tmp = Path(sys.argv[2])


def norm(text: str) -> str:
    lines = []
    for line in text.splitlines():
        if line.startswith("# synced_at:"):
            continue
        lines.append(line)
    return "\n".join(lines) + ("\n" if lines else "")


mismatches = []
for path in sorted((tmp / "conf").rglob("*")):
    if not path.is_file():
        continue
    if any(part.startswith(".") for part in path.relative_to(tmp / "conf").parts):
        continue
    if path.suffix not in {".yaml", ".yml"}:
        continue
    rel = path.relative_to(tmp / "conf")
    real = root / "conf" / rel
    if not real.is_file():
        mismatches.append(f"+ {rel}")
        continue
    if norm(path.read_text(encoding="utf-8")) != norm(real.read_text(encoding="utf-8")):
        mismatches.append(str(rel))
if mismatches:
    print("check_conf_sync: conf/ drift after sync:", file=sys.stderr)
    for m in mismatches[:20]:
        print(f"  {m}", file=sys.stderr)
    sys.exit(1)
print("OK: conf/ in sync")
PY
