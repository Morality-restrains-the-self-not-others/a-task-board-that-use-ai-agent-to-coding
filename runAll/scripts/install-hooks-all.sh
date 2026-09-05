#!/usr/bin/env bash
# 统一 Git Hooks 激活入口（git-hooks-version-control v13）—
# 遍历 meta-repo 主仓 + 全部子仓：core.hooksPath 指向各仓入库的 .githooks/。
#
# 设计：docs/superpowers/specs/2026-08-06-git-hooks-version-control-design.md
# 用法（meta-repo 根）：
#   bash runAll/scripts/install-hooks-all.sh           # 全仓幂等激活
#   bash runAll/scripts/install-hooks-all.sh --check   # 仅校验并报告（不修改）
#   bash runAll/scripts/install-hooks-all.sh taskAuth  # 仅处理指定子仓
set -euo pipefail

ROOT="$(cd "$(git rev-parse --show-toplevel)" && pwd)"
CHECK_ONLY=0
TARGETS=("$@")

if [ "${TARGETS[0]:-}" = "--check" ]; then
  CHECK_ONLY=1
  TARGETS=("${TARGETS[@]:1}")
fi

declare -a REPOS=("$ROOT")
while IFS= read -r path; do
  [ -n "$path" ] && REPOS+=("$ROOT/$path")
done < <(git -C "$ROOT" submodule --quiet foreach 'echo "$sm_path"')

[ ${#TARGETS[@]} -gt 0 ] && REPOS=("${TARGETS[@]}")

status_all=0
for repo in "${REPOS[@]}"; do
  [ -d "$repo/.git" ] || [ -f "$repo/.git" ] || { echo "SKIP not a git repo: $repo"; continue; }
  name="$(basename "$repo")"
  if [ ! -d "$repo/.githooks" ]; then
    echo "⚠ $name: no .githooks/ dir (尚未部署钩子源)"
    status_all=1
    continue
  fi
  hooks_path="$(git -C "$repo" config core.hooksPath 2>/dev/null || true)"
  if [ "$CHECK_ONLY" -eq 1 ]; then
    if [ "$hooks_path" = ".githooks" ]; then
      echo "✅ $name: hooksPath=.githooks"
    else
      echo "❌ $name: hooksPath=${hooks_path:-<unset>}"
      status_all=1
    fi
    continue
  fi
  if [ "$hooks_path" = ".githooks" ]; then
    echo "✅ $name: already active"
  else
    git -C "$repo" config core.hooksPath .githooks
    echo "⚙  $name: hooksPath set → .githooks"
  fi
done

if [ "$status_all" -ne 0 ]; then
  echo ""
  echo "⚠ 存在未激活/未部署的仓库 — 部署钩子源请运行："
  echo "   bash scripts/deploy_repo_random_precommit.sh   # 或 commit_with_submodules.py --deploy-hooks"
fi
exit "$status_all"
