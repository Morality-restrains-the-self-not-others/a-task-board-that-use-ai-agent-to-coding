#!/usr/bin/env bash
# 主仓 hooks 激活校验（git-hooks-version-control v13）— 复制模式已退役。
#
# 背景：.githooks/ 是受版本管理的根仓门禁源，由 Git 原生 core.hooksPath 激活。
#   - commit-msg  : bug-fix 提交必须携带对应回归单测（.ai/01_project_constraints/41_bug_fix_unit_test_required.md）
#   - pre-commit  : 子仓库优先提交门禁（32_submodule_commit_order.md）
#   - pre-push    : 推送前检查
# .git/hooks/ 不再放置业务钩子副本（v13 起）；本脚本仅校验激活状态与残留副本。
#
# 用法（monorepo 根）：
#   bash runAll/scripts/install_root_hooks.sh          # 校验并提示
#   bash runAll/scripts/install_root_hooks.sh --fix    # 自动设置 hooksPath + 清理副本
# 由 commit_with_submodules.py --deploy-hooks / --apply 自动调用。
set -euo pipefail

ROOT="$(cd "$(git rev-parse --show-toplevel 2>/dev/null || echo "$(dirname "$0")/../..")" && pwd)"
FIX="${1:-}"

SRC="$ROOT/.githooks"
[ -d "$SRC" ] || { echo "⚠ .githooks/ not found at $ROOT — root hooks skipped"; exit 0; }

GIT_DIR="$(git -C "$ROOT" rev-parse --git-dir 2>/dev/null || echo "$ROOT/.git")"
[[ "$GIT_DIR" != /* ]] && GIT_DIR="$ROOT/$GIT_DIR"
DST="$GIT_DIR/hooks"

hooks_path="$(git -C "$ROOT" config core.hooksPath 2>/dev/null || true)"
if [ "$hooks_path" = ".githooks" ]; then
  echo "✅ root hooksPath=.githooks (OK)"
else
  echo "❌ root hooksPath=${hooks_path:-<unset>} — expected .githooks"
  if [ "$FIX" = "--fix" ]; then
    git -C "$ROOT" config core.hooksPath .githooks
    echo "⚙  set core.hooksPath .githooks"
  else
    echo "   修复: bash runAll/scripts/install_root_hooks.sh --fix"
    exit 1
  fi
fi

# 残留副本检查（v13 起 .git/hooks/ 仅允许 *.sample）
stale=0
for f in "$DST"/commit-msg "$DST"/pre-commit "$DST"/pre-push "$DST"/check_bug_fix_commit_msg.sh; do
  [ -f "$f" ] || continue
  echo "⚠ 残留副本: $f"
  stale=$((stale + 1))
  if [ "$FIX" = "--fix" ]; then
    rm -f "$f"
    echo "   removed"
  fi
done
[ "$stale" -eq 0 ] && echo "✅ .git/hooks/ 无业务副本残留"
exit 0
