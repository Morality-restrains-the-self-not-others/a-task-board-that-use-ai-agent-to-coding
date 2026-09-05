#!/usr/bin/env bash
set -euo pipefail
ROOT="$(cd "$(dirname "$0")/../../.." && pwd)"
SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"

# Step 1: Run the standard conf-sync for any sync.manifest.yaml fragments
if [ -f "$SCRIPT_DIR/sync.manifest.yaml" ]; then
  python3 "$ROOT/runAll/scripts/conf-sync.py" auth/task-credential
fi

# Step 2: Copy gitOauth provider configs (no cross-service direct reads at runtime)
PROVIDERS_SRC="$ROOT/conf/auth/git-oauth/providers"
PROVIDERS_DST="$SCRIPT_DIR/git-oauth-providers"
if [ -d "$PROVIDERS_SRC" ]; then
  rm -rf "$PROVIDERS_DST"
  cp -r "$PROVIDERS_SRC" "$PROVIDERS_DST"
  echo "synced git-oauth-providers/"
fi
