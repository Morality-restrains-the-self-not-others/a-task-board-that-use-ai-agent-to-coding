#!/usr/bin/env bash
# 统一 Hooks 部署器（git-hooks-version-control v13）—
# 将 SSOT 模板（scripts/hooks/templates/）渲染到各子仓 .githooks/（入库位），
# 并以 core.hooksPath 激活。.git/hooks/ 复制模式已退役（不再写入副本）。
#
# 设计: docs/superpowers/specs/2026-08-06-git-hooks-version-control-design.md
# Usage:
#   bash scripts/deploy_repo_random_precommit.sh           # 全部子仓
#   bash scripts/deploy_repo_random_precommit.sh taskAuth  # 指定子仓
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
TEMPLATES="$ROOT/scripts/hooks/templates"
HOOK_VERSION="1.0.0"

# 以 .gitmodules 为 SSOT 动态生成仓库清单（消除硬编码漂移）
mapfile -t ALL_REPOS < <(grep -E '^\s*path = ' "$ROOT/.gitmodules" | sed 's/.*path = //' | tr -d ' ')

pick_template() {
  local repo="$1"
  case "$repo" in
    task2app)
      echo "SKIP" # 已删除
      return
      ;;
    taskAiProvider)
      echo "pre-commit.go_js"
      return
      ;;
    db)
      echo "pre-commit.go_python"
      return
      ;;
    trae-agent)
      echo "pre-commit.python"
      return
      ;;
    conf|dockerInfra|sdk|taskGateway|gitService|taskChromePlugin)
      echo "pre-commit.placeholder"
      return
      ;;
  esac

  local has_go=0 has_py=0 has_js=0
  if find "$ROOT/$repo" -name '*_test.go' -not -path '*/vendor/*' -not -path '*/.git/*' 2>/dev/null | head -1 | grep -q .; then
    has_go=1
  fi
  if find "$ROOT/$repo" \( -name 'test_*.py' -o -name '*_test.py' \) \
      -not -path '*/.git/*' -not -path '*/venv*' -not -path '*/node_modules/*' 2>/dev/null | head -1 | grep -q .; then
    has_py=1
  fi
  if find "$ROOT/$repo" \( -name '*.unit.test.js' -o -name '*.test.js' -o -name '*.test.mjs' \) \
      -not -path '*/node_modules/*' -not -path '*/.git/*' \
      -not -name '*.playwright*' -not -name '*.e2e*' 2>/dev/null | head -1 | grep -q .; then
    has_js=1
  fi

  if [ "$has_go" -eq 1 ] && [ "$has_js" -eq 1 ]; then
    echo "pre-commit.go_js"
  elif [ "$has_go" -eq 1 ] && [ "$has_py" -eq 1 ]; then
    echo "pre-commit.go_python"
  elif [ "$has_go" -eq 1 ]; then
    echo "pre-commit.go"
  elif [ "$has_py" -eq 1 ]; then
    echo "pre-commit.python"
  elif [ "$has_js" -eq 1 ]; then
    echo "pre-commit.node"
  else
    echo "pre-commit.placeholder"
  fi
}

# 共享抽测库分发（SSOT → 各仓 .githooks/lib/）
ensure_lib() {
  local repo="$1"
  mkdir -p "$ROOT/$repo/.githooks/lib"
  cp "$TEMPLATES/lib/random_test_runner.sh" "$ROOT/$repo/.githooks/lib/random_test_runner.sh"
  chmod +x "$ROOT/$repo/.githooks/lib/random_test_runner.sh"
}

# 写通用激活脚本 + 版本清单
write_common() {
  local repo="$1"
  cp "$TEMPLATES/install.sh" "$ROOT/$repo/.githooks/install.sh"
  chmod +x "$ROOT/$repo/.githooks/install.sh"
  cat > "$ROOT/$repo/.githooks/HOOK_VERSION" <<EOF
# Git Hooks 版本清单 — $repo（git-hooks-version-control v13）
pre-commit=$HOOK_VERSION
commit-msg=$HOOK_VERSION
check_bug_fix_commit_msg.sh=$HOOK_VERSION
EOF
}

# commit-msg 门禁（41 号规则：整个 monorepo 生效）— 缺失时从 SSOT 部署
ensure_commit_msg() {
  local repo="$1"
  [ -f "$ROOT/$repo/.githooks/commit-msg" ] && return 0
  cp "$TEMPLATES/commit-msg" "$ROOT/$repo/.githooks/commit-msg"
  cp "$TEMPLATES/check_bug_fix_commit_msg.sh" "$ROOT/$repo/.githooks/check_bug_fix_commit_msg.sh"
  chmod +x "$ROOT/$repo/.githooks/commit-msg" "$ROOT/$repo/.githooks/check_bug_fix_commit_msg.sh"
}

deploy_one() {
  local repo="$1"
  if [ ! -d "$ROOT/$repo" ] || [ ! -e "$ROOT/$repo/.git" ]; then
    echo "SKIP not a git repo: $repo"
    return 0
  fi

  local tmpl
  tmpl="$(pick_template "$repo")"
  if [ "$tmpl" = "SKIP" ]; then
    echo "SKIP $repo (removed)"
    return 0
  fi

  # 已有 .githooks/pre-commit（入库真源）→ 保留定制内容，仅补 lib/install/版本 + 激活
  if [ -f "$ROOT/$repo/.githooks/pre-commit" ]; then
    echo "KEEP existing .githooks/pre-commit: $repo"
    ensure_lib "$repo"
    write_common "$repo"
    ensure_commit_msg "$repo"
    (cd "$ROOT/$repo" && git config core.hooksPath .githooks)
    return 0
  fi

  local src="$TEMPLATES/$tmpl"
  if [ ! -f "$src" ]; then
    echo "ERROR missing template $src" >&2
    return 1
  fi

  mkdir -p "$ROOT/$repo/.githooks"
  cp "$src" "$ROOT/$repo/.githooks/pre-commit"
  chmod +x "$ROOT/$repo/.githooks/pre-commit"
  ensure_lib "$repo"
  write_common "$repo"
  ensure_commit_msg "$repo"
  (cd "$ROOT/$repo" && git config core.hooksPath .githooks)
  echo "DEPLOYED $repo <- $tmpl (HOOK_VERSION $HOOK_VERSION)"
}

TARGETS=("$@")
if [ "${#TARGETS[@]}" -eq 0 ]; then
  TARGETS=("${ALL_REPOS[@]}")
fi

ok=0
fail=0
for repo in "${TARGETS[@]}"; do
  if deploy_one "$repo"; then
    ok=$((ok + 1))
  else
    fail=$((fail + 1))
  fi
done

echo "Done. ok=$ok fail=$fail"
[ "$fail" -eq 0 ]
