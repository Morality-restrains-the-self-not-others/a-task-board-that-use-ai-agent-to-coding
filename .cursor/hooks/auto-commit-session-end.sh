#!/usr/bin/env bash
# Cursor sessionEnd → meta 仓自动提交到 main（与 Claude SessionEnd 对齐）
# stdin 为 Cursor hook JSON，需排空以免 SIGPIPE；提交走完整门禁。
set -u
cat >/dev/null || true
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"
exec bash "$REPO_ROOT/scripts/lib/auto-commit.sh"
