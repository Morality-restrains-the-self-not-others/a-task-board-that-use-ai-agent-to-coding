#!/usr/bin/env bash
# 硬约束 61 复查脚本 — 扫描 git 历史重写遗留标记（只读，不修改任何内容）
#
# 检测标记（与 .githooks/pre-commit v1.4.0 门禁同源）：
#   - .git/filter-repo/      git filter-repo 运行后必然留下
#   - refs/original/*        git filter-branch 备份 ref
#   - refs/replace/*         git replace 重写覆盖 ref
#
# 用法：
#   bash scripts/lib/check_git_history_rewrite.sh --scan [repo-dir...]  # 默认当前仓库；多目录逐个扫描
#   bash scripts/lib/check_git_history_rewrite.sh --scan-all            # meta + 全部 .gitmodules 登记子仓
# 退出码：0 = 全部干净；1 = 存在标记（或目录非法）
set -euo pipefail

SCAN_ALL=0
REPOS=()

while [ $# -gt 0 ]; do
  case "$1" in
    --scan-all) SCAN_ALL=1 ;;
    --scan) : ;;
    -*) echo "未知参数: $1" >&2; echo "用法: $0 [--scan-all] | --scan [repo-dir...]" >&2; exit 2 ;;
    *) REPOS+=("$1") ;;
  esac
  shift
done

META_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"

if [ "$SCAN_ALL" -eq 1 ]; then
  REPOS=("$META_ROOT")
  if [ -f "$META_ROOT/.gitmodules" ]; then
    while IFS= read -r sub; do
      [ -n "$sub" ] && REPOS+=("$sub")
    done < <(git config --file "$META_ROOT/.gitmodules" --get-regexp '^submodule\..*\.path$' 2>/dev/null | awk '{print $2}')
  fi
fi
if [ ${#REPOS[@]} -eq 0 ]; then
  REPOS=("$PWD")
fi

_check_repo() {
  local repo="$1"
  [ -d "$repo/.git" ] || [ -f "$repo/.git" ] || { echo "⚠ [skip] 非 git 仓库或未克隆: $repo"; return 0; }
  local gcd
  gcd="$(git -C "$repo" rev-parse --git-common-dir 2>/dev/null || echo "$repo/.git")"
  case "$gcd" in
    /*) : ;;
    *) gcd="$repo/$gcd" ;;
  esac
  local found=0
  [ -d "$gcd/filter-repo" ] && { echo "✗ [$repo] $gcd/filter-repo/（git filter-repo 遗留标记）"; found=1; }
  if [ -n "$(git -C "$repo" for-each-ref --format='%(refname)' refs/original 2>/dev/null)" ]; then
    echo "✗ [$repo] refs/original/* 备份 ref（git filter-branch 遗留标记）"; found=1
  fi
  if [ -n "$(git -C "$repo" for-each-ref --format='%(refname)' refs/replace 2>/dev/null)" ]; then
    echo "✗ [$repo] refs/replace/* ref（git replace 重写标记）"; found=1
  fi
  if [ "$found" -eq 0 ]; then
    echo "✓ [$repo] 无历史重写遗留标记"
  fi
  return "$found"
}

rc=0
for repo in "${REPOS[@]}"; do
  _check_repo "$repo" || rc=1
done

if [ "$rc" -eq 1 ]; then
  echo ""
  echo "⚠ 检测到 git 历史重写遗留标记。处置见 .ai/01_project_constraints/61_no_git_history_rewrite.md："
  echo "  凭据废弃 + 重新生成（禁止清历史）；仅当历史重写确已发生且全员接受当前基线时，"
  echo "  人工确认后清理标记（rm -rf <git-common-dir>/filter-repo；删除 refs/original/*、refs/replace/*）"
fi
exit "$rc"
