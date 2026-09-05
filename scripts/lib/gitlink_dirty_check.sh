#!/usr/bin/env bash
# 子仓「已跟踪未提交变更」判定（OPT-20260823-041）。
#
# 并行会话可能在工作区留下纯未跟踪文件（??），这些文件既不属于本会话也不该
# 被本会话 git add；若把「仅未跟踪」也当成 dirty，会卡住无关 gitlink 指针同步
# 到已推送 HEAD。本函数只把「存在已跟踪未提交变更」（ M/M /A /D /R /C 等）
# 视为 dirty，纯 ?? 不阻断。
set -euo pipefail

# is_subrepo_tracked_dirty <repo-path>
# 返回 0 表示子仓含已跟踪未提交变更；返回 1 表示 clean 或仅含未跟踪文件。
# 子仓路径不存在或非 git 仓库时返回 1（不阻断，交由登记/克隆流程处理）。
is_subrepo_tracked_dirty() {
  local repo="$1"
  if ! git -C "$repo" rev-parse --git-dir >/dev/null 2>&1; then
    return 1
  fi
  [ -n "$(git -C "$repo" status --porcelain 2>/dev/null | grep -v '^??')" ]
}
