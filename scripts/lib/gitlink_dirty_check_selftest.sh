#!/usr/bin/env bash
# gitlink_dirty_check.sh 自测（OPT-20260823-041）：纯未跟踪文件不判定 dirty，
# 已跟踪未提交变更判定 dirty。模拟 pre-commit MISSING 循环的调用形态。
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
TMP="$(mktemp -d)"
trap 'rm -rf "$TMP"' EXIT

# 模拟子仓
git init -q "$TMP/sub"
git -C "$TMP/sub" config user.email t@example.com
git -C "$TMP/sub" config user.name t
echo 'v1' > "$TMP/sub/tracked.go"
git -C "$TMP/sub" add tracked.go
git -C "$TMP/sub" commit -qm 'init'

# shellcheck source=gitlink_dirty_check.sh
source "$SCRIPT_DIR/gitlink_dirty_check.sh"

# Case 1: 仅未跟踪文件 → 不 dirty（返回 1）
mkdir -p "$TMP/sub/intents"
echo 'untracked-wip' > "$TMP/sub/intents/wip.md"
if is_subrepo_tracked_dirty "$TMP/sub"; then
  echo "FAIL: only-untracked subrepo must NOT be flagged dirty" >&2
  echo "--- sub status ---"; git -C "$TMP/sub" status --porcelain >&2
  exit 1
fi

# Case 2: 已跟踪文件被修改 → dirty（返回 0）
echo 'v2 dirty' > "$TMP/sub/tracked.go"
if ! is_subrepo_tracked_dirty "$TMP/sub"; then
  echo "FAIL: tracked modification must be flagged dirty" >&2
  exit 1
fi

# Case 3: 已跟踪文件新增（staged A）→ dirty
git -C "$TMP/sub" checkout -q -- tracked.go
echo 'new file' > "$TMP/sub/added.go"
git -C "$TMP/sub" add added.go
if ! is_subrepo_tracked_dirty "$TMP/sub"; then
  echo "FAIL: staged addition must be flagged dirty" >&2
  exit 1
fi

# Case 4: 已跟踪删除（D）→ dirty
git -C "$TMP/sub" add tracked.go
git -C "$TMP/sub" rm -q --cached tracked.go
if ! is_subrepo_tracked_dirty "$TMP/sub"; then
  echo "FAIL: tracked deletion must be flagged dirty" >&2
  exit 1
fi

# Case 5: 非 git 目录 → 不 dirty（返回 1）
if is_subrepo_tracked_dirty "$TMP/no-such-repo"; then
  echo "FAIL: non-git dir must not be flagged dirty" >&2
  exit 1
fi

echo "OK: gitlink_dirty_check selftest passed"
