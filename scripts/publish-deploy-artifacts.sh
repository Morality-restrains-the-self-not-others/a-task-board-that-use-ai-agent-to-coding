#!/usr/bin/env bash
# Publish $BIN_DIR files as a GitHub Release tag (artifact plane for ADR-0052).
# First arg is the Release *tag* (e.g. deploy-20260831), not a content SHA.
# GitHub has no GitLab-style generic registry; Release assets are the pin target.
set -euo pipefail
SHA="${1:?usage: publish-deploy-artifacts.sh <release-tag> [bin-dir] [repo]}"
BIN_DIR="${2:-./bin}"
REPO="${3:-task2money/daydaymoney-deploy}"
if [[ ! -d "$BIN_DIR" ]]; then
  echo "publish-deploy-artifacts: missing bin dir $BIN_DIR" >&2
  exit 1
fi
# shellcheck disable=SC2046
gh release create "$SHA" $(find "$BIN_DIR" -maxdepth 1 -type f ! -name '*.sha') \
  --repo "$REPO" \
  --title "deploy $SHA" \
  --notes "Pinned binaries for deploy-sync. Do not attach secrets."
