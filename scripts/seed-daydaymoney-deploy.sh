#!/usr/bin/env bash
# Seed daydaymoney-deploy: sanitised conf + source-less recipes (ADR-0052).
# Does not modify the live conf/ submodule. Does not copy secrets or binaries.
set -euo pipefail

META_ROOT="$(cd "$(dirname "$0")/.." && pwd)"
CONF_SRC="${CONF_SRC:-$META_ROOT/conf}"
DEST="${1:-$META_ROOT/.daydaymoney-deploy-seed}"

if [[ ! -d "$CONF_SRC" ]]; then
  echo "seed-daydaymoney-deploy: missing CONF_SRC=$CONF_SRC" >&2
  exit 1
fi

mkdir -p "$DEST/envs/current/conf"
rsync -a \
  --exclude '.git/' \
  --exclude 'config.local.yaml' \
  --exclude '*.local.yaml' \
  --exclude '.env' \
  --exclude '.env.*' \
  --exclude '*.pem' \
  --exclude '*.key' \
  --exclude '*Pwd.md' \
  --exclude '*pwd.md' \
  "$CONF_SRC/" "$DEST/envs/current/conf/"

if [[ ! -f "$DEST/envs/current/releases.yaml" && -f "$META_ROOT/conf.example/releases.yaml" ]]; then
  cp "$META_ROOT/conf.example/releases.yaml" "$DEST/envs/current/releases.yaml"
fi

META_ROOT="$META_ROOT" bash "$META_ROOT/runAll/scripts/export-deploy-payload.sh" "$DEST"

if [[ ! -f "$DEST/README.md" ]] || ! grep -q 'scripts/up.sh' "$DEST/README.md"; then
  echo "seed-daydaymoney-deploy: refresh README.md"
  cat > "$DEST/README.md" <<'EOF'
# daydaymoney-deploy

Private runtime repo (ADR-0052): clone and run without the source monorepo.

See `scripts/up.sh`. Host secrets stay in `conf-local/` (never commit).
EOF
fi

echo "seed-daydaymoney-deploy: wrote $DEST"
