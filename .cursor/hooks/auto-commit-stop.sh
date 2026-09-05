#!/usr/bin/env bash
# Cursor stop → 低频检查点自动提交到 main（阈值 30 分钟，对齐 Claude Stop）
set -u
cat >/dev/null || true
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"
exec bash "$REPO_ROOT/scripts/lib/auto-commit.sh" --checkpoint-threshold 1800
