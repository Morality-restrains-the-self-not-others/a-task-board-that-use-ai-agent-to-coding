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
HOOK_VERSION="1.1.0"

# 真定制 pre-commit 仓清单（非模板派生物 — 升级时仅注入锁校验，不覆盖）
# docs: 架构视图自动归档钩子；taskChromePlugin: 随机单测 + Playwright 登录 E2E
CUSTOM_HOOK_REPOS="docs taskChromePlugin"
# 配置/依赖仓：不部署模板钩子 / .claude/settings.json；仅保留手写 pre-commit
# conf: 手写 oauth live-check pre-commit；sdk: 第三方 vendored SDK（无自有测例可跑）
SKIP_HOOK_REPOS="conf sdk"

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
    taskFE)
      # 前端仓：JS 单测为主体；tests/helpers/test_*.py 仅为 E2E seed 辅助器（OPT-20260815-009）
      echo "pre-commit.node"
      return
      ;;
    # OPT-20260901-022: 去掉 dockerInfra/sdk/taskGateway/gitService/taskChromePlugin 的
    # 占位硬编码——这些仓均有真实测例（dockerInfra config 测例、taskGateway pytest、
    # gitService Go+py、taskChromePlugin JS 单测），走下方探测拿真实模板。
    # sdk 为第三方 vendored SDK，已纳入 SKIP_HOOK_REPOS。
  esac

  # 探测禁止 find|head+pipefail（SIGPIPE 会误判无测例，OPT-20260815-009）；
  # 用 -print -quit 找到首个匹配即退出，无 SIGPIPE
  local has_go=0 has_py=0 has_js=0
  if find "$ROOT/$repo" -name '*_test.go' -not -path '*/vendor/*' -not -path '*/.git/*' -print -quit 2>/dev/null | grep -q .; then
    has_go=1
  fi
  if find "$ROOT/$repo" \( -name 'test_*.py' -o -name '*_test.py' \) \
      -not -path '*/.git/*' -not -path '*/venv*' -not -path '*/node_modules/*' -print -quit 2>/dev/null | grep -q .; then
    has_py=1
  fi
  if find "$ROOT/$repo" \( -name '*.unit.test.js' -o -name '*.test.js' -o -name '*.test.mjs' \) \
      -not -path '*/node_modules/*' -not -path '*/.git/*' \
      -not -name '*.playwright*' -not -name '*.e2e*' -print -quit 2>/dev/null | grep -q .; then
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
  # v1.1.0: Session Hub 提交锁校验库
  cp "$TEMPLATES/lib/session_lock_check.sh" "$ROOT/$repo/.githooks/lib/session_lock_check.sh"
  chmod +x "$ROOT/$repo/.githooks/lib/session_lock_check.sh"
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
session_lock_check.sh=$HOOK_VERSION
EOF
}

# 会话钩子 settings.json 分发（SSOT 模板 → 各仓 .claude/settings.json，机器本地配置）
# 渲染 @META_ROOT@ 占位符为实际 meta root 路径；内容一致则跳过（幂等）
deploy_settings() {
  local repo="$1"
  local target="$ROOT/$repo/.claude/settings.json"
  mkdir -p "$(dirname "$target")"
  if [ -f "$target" ] && cmp -s <(sed "s|@META_ROOT@|$ROOT|g" "$TEMPLATES/claude-settings.json") "$target"; then
    echo "  settings.json: KEEP $repo（已含最新会话钩子）"
  else
    sed "s|@META_ROOT@|$ROOT|g" "$TEMPLATES/claude-settings.json" > "$target"
    echo "  settings.json: DEPLOYED $repo"
  fi
  # 机器本地配置（渲染后含绝对路径）不入库：.git/info/exclude 排除（root 为入库 SSOT 除外）
  # --git-path 对独立 .git 目录返回相对路径（相对仓根）；submodule gitdir 可能尚无 exclude — 先确保父目录与文件
  if [ "$repo" != "." ]; then
    local exclude
    exclude="$(git -C "$ROOT/$repo" rev-parse --git-path info/exclude 2>/dev/null || true)"
    case "$exclude" in
      /*) ;;
      *) exclude="$ROOT/$repo/$exclude" ;;
    esac
    if [ -n "$exclude" ]; then
      mkdir -p "$(dirname "$exclude")"
      touch "$exclude"
      if ! grep -q "^\.claude/settings\.json$" "$exclude"; then
        printf '\n# claude-code session hooks (machine-local; deploy via deploy_repo_random_precommit.sh)\n.claude/settings.json\n' >> "$exclude"
      fi
    fi
  fi
}

# v1.1.0 锁校验注入（真定制 pre-commit 保留原逻辑，仅补 Session Hub 提交锁校验块）
inject_lock_check() {
  local pc="$1"
  if grep -q "session_hub_lock_check" "$pc"; then
    return 0
  fi
  python3 - "$pc" <<'PYEOF'
import re, sys
path = sys.argv[1]
text = open(path, encoding="utf-8").read()
block = (
    "\n# Session Hub 提交锁校验（v1.1.0，可选 — 二进制缺失时放行）\n"
    'SES_LIB="$(cd "$(dirname "$0")" && pwd)/lib/session_lock_check.sh"\n'
    'if [ -f "$SES_LIB" ]; then source "$SES_LIB"; session_hub_lock_check; fi\n'
)
m = re.search(r"^set -[a-z]*u[a-z ]*$", text, re.M)
if m:
    text = text[: m.end()] + block + text[m.end():]
else:
    text = text.rstrip("\n") + "\n" + block
open(path, "w", encoding="utf-8").write(text)
PYEOF
  chmod +x "$pc"
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

  if echo "$SKIP_HOOK_REPOS" | grep -qw "$repo"; then
    # OPT-20260901-022: 清理历史误灌的模板派生物，仅保留手写 pre-commit
    # （conf 的 oauth live-check 等）。占位 pre-commit 也是模板派生物，一并删除。
    local gd="$ROOT/$repo/.githooks"
    rm -f "$gd/commit-msg" "$gd/check_bug_fix_commit_msg.sh" "$gd/install.sh" "$gd/HOOK_VERSION"
    rm -rf "$gd/lib"
    if [ -f "$gd/pre-commit" ] && grep -q "无测例仓库占位\|当前无自动化单元测例" "$gd/pre-commit"; then
      rm -f "$gd/pre-commit"
    fi
    echo "SKIP_HOOK $repo (config/vendored repo; template hooks removed, hand-written kept)"
    return 0
  fi

  local tmpl
  tmpl="$(pick_template "$repo")"
  if [ "$tmpl" = "SKIP" ]; then
    echo "SKIP $repo (removed)"
    return 0
  fi

  local src="$TEMPLATES/$tmpl"
  if [ ! -f "$src" ]; then
    echo "ERROR missing template $src" >&2
    return 1
  fi

  # 已有 .githooks/pre-commit（入库真源）→ 升级策略：
  #   v1.1.0（含锁校验）→ 保留；模板派生物（旧版/占位）→ 覆盖为当前模板；
  #   真定制（CUSTOM_HOOK_REPOS）→ 仅注入锁校验块，保留原逻辑
  if [ -f "$ROOT/$repo/.githooks/pre-commit" ]; then
    # OPT-20260815-009: 已含锁校验的钩子仍要校验模板类型匹配 —
    # 历史误判（find|head SIGPIPE）写入的占位/错误类型钩子不得因含锁校验而永久 KEEP。
    # 模板特征: "Running pre-commit <Lang> tests"（占位钩子无此特征）
    _matches_template() {
      case "$tmpl" in
        pre-commit.placeholder) return 1 ;;  # 占位目标无特征，交给下方占位分支判断
        pre-commit.go)          grep -q "Running pre-commit Go tests" "$ROOT/$repo/.githooks/pre-commit" ;;
        pre-commit.node)        grep -q "Running pre-commit Node tests" "$ROOT/$repo/.githooks/pre-commit" ;;
        pre-commit.python)      grep -q "Running pre-commit Python tests" "$ROOT/$repo/.githooks/pre-commit" ;;
        pre-commit.go_js)       grep -qE "Running pre-commit (Go|Node) tests" "$ROOT/$repo/.githooks/pre-commit" ;;
        pre-commit.go_python)   grep -qE "Running pre-commit (Go|Python) tests" "$ROOT/$repo/.githooks/pre-commit" ;;
        *) return 0 ;;
      esac
    }
    if grep -q "session_hub_lock_check" "$ROOT/$repo/.githooks/pre-commit"; then
      if [ "$tmpl" != "pre-commit.placeholder" ] && ! _matches_template; then
        echo "UPGRADE $repo <- $tmpl (钩子类型与探测结果不符 → 重新部署)"
        cp "$src" "$ROOT/$repo/.githooks/pre-commit"
        chmod +x "$ROOT/$repo/.githooks/pre-commit"
        inject_lock_check "$ROOT/$repo/.githooks/pre-commit"
      elif [ "$tmpl" = "pre-commit.placeholder" ] \
          && grep -q "当前无自动化单元测例\|无测例仓库占位" "$ROOT/$repo/.githooks/pre-commit"; then
        echo "KEEP existing .githooks/pre-commit: $repo (占位钩子，探测确认无测例)"
      else
        echo "KEEP existing .githooks/pre-commit: $repo (v1.1.0 含锁校验且类型匹配)"
      fi
    elif echo "$CUSTOM_HOOK_REPOS" | grep -qw "$repo"; then
      echo "INJECT session lock check: $repo (定制 pre-commit 保留)"
      inject_lock_check "$ROOT/$repo/.githooks/pre-commit"
    else
      echo "UPGRADE $repo <- $tmpl (旧模板派生物 → v1.1.0)"
      cp "$src" "$ROOT/$repo/.githooks/pre-commit"
      chmod +x "$ROOT/$repo/.githooks/pre-commit"
    fi
    ensure_lib "$repo"
    write_common "$repo"
    ensure_commit_msg "$repo"
    (cd "$ROOT/$repo" && git config core.hooksPath .githooks)
    deploy_settings "$repo"
    return 0
  fi

  mkdir -p "$ROOT/$repo/.githooks"
  cp "$src" "$ROOT/$repo/.githooks/pre-commit"
  chmod +x "$ROOT/$repo/.githooks/pre-commit"
  ensure_lib "$repo"
  write_common "$repo"
  ensure_commit_msg "$repo"
  (cd "$ROOT/$repo" && git config core.hooksPath .githooks)
  deploy_settings "$repo"
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

# meta root 自身 settings.json 与 SSOT 同步（root 不参与子仓循环）
deploy_settings "."

echo "Done. ok=$ok fail=$fail"
[ "$fail" -eq 0 ]
