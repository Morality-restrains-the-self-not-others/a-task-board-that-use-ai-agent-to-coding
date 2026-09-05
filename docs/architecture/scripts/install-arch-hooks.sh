#!/usr/bin/env bash
# docs 子模块 hooks 激活校验（git-hooks-version-control v13）— 复制模式已退役。
#
# 背景：docs/.githooks/ 是受版本管理的 hook 真源（入库可跟踪），由 Git 原生
#       core.hooksPath 激活 — 零复制、零漂移。.git/hooks/ 不再放置业务钩子副本。
#       docs 是 git submodule，提交发生在子模块内，meta-repo 的 install_root_hooks.sh
#       不覆盖子模块 —— 本脚本负责 docs 自身的激活校验。
#
# 用法（docs 仓库根 或 meta-repo 根均可）：
#   bash architecture/scripts/install-arch-hooks.sh          # 校验并提示
#   bash architecture/scripts/install-arch-hooks.sh --fix    # 自动设置 hooksPath + 清理副本
set -euo pipefail

ROOT="$(cd "$(git rev-parse --show-toplevel 2>/dev/null || echo "$(dirname "$0")/../..")" && pwd)"
FIX="${1:-}"

SRC="$ROOT/.githooks"
if [ ! -d "$SRC" ]; then
  echo "⚠ .githooks/ not found at $ROOT — arch hooks skipped"
  exit 0
fi

GIT_DIR="$(git -C "$ROOT" rev-parse --git-dir 2>/dev/null || echo "$ROOT/.git")"
[[ "$GIT_DIR" != /* ]] && GIT_DIR="$ROOT/$GIT_DIR"
DST="$GIT_DIR/hooks"

hooks_path="$(git -C "$ROOT" config core.hooksPath 2>/dev/null || true)"
if [ "$hooks_path" = ".githooks" ]; then
  echo "✅ docs hooksPath=.githooks (OK)"
else
  echo "❌ docs hooksPath=${hooks_path:-<unset>} — expected .githooks"
  if [ "$FIX" = "--fix" ]; then
    git -C "$ROOT" config core.hooksPath .githooks
    echo "⚙  set core.hooksPath .githooks"
  else
    echo "   修复: bash architecture/scripts/install-arch-hooks.sh --fix"
    exit 1
  fi
fi

# 残留副本检查（v13 起 .git/hooks/ 仅允许 *.sample）
stale=0
for f in "$DST"/pre-commit "$DST"/commit-msg "$DST"/pre-push "$DST"/check_bug_fix_commit_msg.sh; do
  [ -f "$f" ] || continue
  echo "⚠ 残留副本: $f"
  stale=$((stale + 1))
  if [ "$FIX" = "--fix" ]; then
    rm -f "$f"
    echo "   removed"
  fi
done
[ "$stale" -eq 0 ] && echo "✅ docs .git/hooks/ 无业务副本残留"
exit 0
