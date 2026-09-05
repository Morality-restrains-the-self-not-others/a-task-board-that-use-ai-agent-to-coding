#!/usr/bin/env bash
# ============================================================================
# auto-commit_selftest.sh — auto-commit.sh 功能自测（scratch repo）
# ============================================================================
# 覆盖：
#   1) 已在 main → 有变更时成功提交到 main
#   2) 在非 main 分支 → 自动切换到 main 再提交
#   3) detached HEAD → 跳过（不提交）
#   4) 项目 Cursor hooks.json 已注册 sessionEnd/stop → auto-commit
#   5) 并行进程已提交竞态 → 不误报 FAILED（OPT-20260810-019）
#
# 用法: bash scripts/lib/auto-commit_selftest.sh
# 已挂载 CI: 根仓 .pre-commit-config.yaml hook `check-auto-commit-selftest`
# （OPT-20260810-018），每次 meta 提交自动执行。
# 预期耗时: ~7s（4 个 scratch repo 依次跑 auto-commit.sh）。
# scratch 清理: 各用例用 mktemp -d /tmp/auto-commit-selftest.XXXXXX 创建，
#   用例结束即 rm -rf；中断残留可用 `rm -rf /tmp/auto-commit-selftest.*` 清除。
# ============================================================================
set -u

# git ≥2.53 在 meta `git commit` 时导出 GIT_INDEX_FILE=<gitdir>/next-index-*.lock；
# 自测在 scratch repo 运行嵌套 git 命令，若继承该变量会误用 meta 索引而失败，
# 故此处强制清空（auto-commit.sh 同理只应操作目标仓自己的索引）。
unset GIT_INDEX_FILE

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
AUTO_COMMIT="$SCRIPT_DIR/auto-commit.sh"
META_ROOT="$(git -C "$SCRIPT_DIR" rev-parse --show-toplevel)"
PASS=0
FAIL=0

assert_eq() {
  local name="$1" got="$2" want="$3"
  if [ "$got" = "$want" ]; then
    echo "  PASS: $name"
    PASS=$((PASS + 1))
  else
    echo "  FAIL: $name (got='$got' want='$want')" >&2
    FAIL=$((FAIL + 1))
  fi
}

assert_true() {
  local name="$1"
  shift
  if "$@"; then
    echo "  PASS: $name"
    PASS=$((PASS + 1))
  else
    echo "  FAIL: $name" >&2
    FAIL=$((FAIL + 1))
  fi
}

make_scratch() {
  local dir
  dir="$(mktemp -d /tmp/auto-commit-selftest.XXXXXX)"
  git -C "$dir" init -b main >/dev/null 2>&1
  git -C "$dir" config user.email "selftest@example.com"
  git -C "$dir" config user.name "auto-commit-selftest"
  # 禁用门禁钩子，避免 scratch 触发 meta 仓 pre-commit
  mkdir -p "$dir/.empty-hooks"
  git -C "$dir" config core.hooksPath "$dir/.empty-hooks"
  # 初始提交，使 main 存在且可切换
  echo "base" > "$dir/README"
  git -C "$dir" add README
  git -C "$dir" commit -m "init" >/dev/null
  echo "$dir"
}

run_auto_commit() {
  local repo="$1"
  shift
  AUTO_COMMIT_REPO_ROOT="$repo" bash "$AUTO_COMMIT" "$@" 2>&1
}

echo "== auto-commit self-test =="

# ── 1) 已在 main ─────────────────────────────────────────────────
echo "[1] commit on main"
SCRATCH="$(make_scratch)"
echo "change" > "$SCRATCH/file1.txt"
OUT="$(run_auto_commit "$SCRATCH" || true)"
assert_eq "branch still main" "$(git -C "$SCRATCH" branch --show-current)" "main"
assert_true "file1 committed" git -C "$SCRATCH" cat-file -e HEAD:file1.txt
assert_true "message contains auto-commit" \
  bash -c "git -C '$SCRATCH' log -1 --pretty=%s | grep -q 'auto-commit'"
rm -rf "$SCRATCH"

# ── 2) 非 main → 切到 main 再提交 ────────────────────────────────
echo "[2] switch from feature to main then commit"
SCRATCH="$(make_scratch)"
git -C "$SCRATCH" switch -c feature/wip >/dev/null
echo "wip" > "$SCRATCH/wip.txt"
OUT="$(run_auto_commit "$SCRATCH" || true)"
assert_eq "landed on main" "$(git -C "$SCRATCH" branch --show-current)" "main"
assert_true "wip.txt on main HEAD" git -C "$SCRATCH" cat-file -e HEAD:wip.txt
assert_true "log mentions switch or auto-commit" \
  bash -c "echo '$OUT' | grep -Eq '切换到 main|auto-commit:'"
# feature 分支不应因本次自动提交前进（提交发生在 main）
FEATURE_HAS="$(git -C "$SCRATCH" log feature/wip --oneline -- wip.txt 2>/dev/null | wc -l)"
assert_eq "feature branch has no wip commit" "$(echo "$FEATURE_HAS" | tr -d ' ')" "0"
rm -rf "$SCRATCH"

# ── 3) detached HEAD → skip ──────────────────────────────────────
echo "[3] skip detached HEAD"
SCRATCH="$(make_scratch)"
SHA="$(git -C "$SCRATCH" rev-parse HEAD)"
git -C "$SCRATCH" checkout --detach "$SHA" >/dev/null 2>&1
echo "detached-change" > "$SCRATCH/detached.txt"
OUT="$(run_auto_commit "$SCRATCH" || true)"
assert_true "mentions detached skip" \
  bash -c "echo '$OUT' | grep -Eq 'detached HEAD|跳过'"
assert_true "detached.txt still untracked/uncommitted" \
  bash -c "git -C '$SCRATCH' status --porcelain | grep -q 'detached.txt'"
rm -rf "$SCRATCH"

# ── 5) 并行进程已提交 → 不误报 FAILED（OPT-20260810-019）──────────
echo "[5] race: parallel session committed → no false FAILED"
SCRATCH="$(make_scratch)"
# pre-commit 钩子先以「并行会话」身份把暂存内容提交掉（--no-verify），再让本进程 commit 失败。
# 模拟 Stop/SessionEnd 竞态：另一进程已推进 HEAD，工作区相对 HEAD 已干净。
mkdir -p "$SCRATCH/race-hooks"
cat > "$SCRATCH/race-hooks/pre-commit" <<'EOF'
#!/bin/bash
git commit --no-verify -m "parallel-race-commit" >/dev/null 2>&1
exit 1
EOF
chmod +x "$SCRATCH/race-hooks/pre-commit"
git -C "$SCRATCH" config core.hooksPath "$SCRATCH/race-hooks"
echo "race-change" > "$SCRATCH/race.txt"
OUT="$(run_auto_commit "$SCRATCH" || true)"
assert_true "no FAILED when tree clean" \
  bash -c "! echo '$OUT' | grep -q 'FAILED'"
assert_true "reports committed by other session" \
  bash -c "echo '$OUT' | grep -q '已被其他会话提交'"
assert_true "tree clean after race" \
  bash -c "[ -z \"\$(git -C '$SCRATCH' status --porcelain)\" ]"
rm -rf "$SCRATCH"

# ── 6) 嵌套 git 目录不得被收成 gitlink（OPT-20260815-007）─────────
echo "[6] nested git dir must not become gitlink"
SCRATCH="$(make_scratch)"
# 在 scratch 里放一个自带 .git 的目录（模拟 *-wt worktree）；scratch 无 .gitignore 时会命中 git add -A
git -C "$SCRATCH" init -b main "$SCRATCH/foo-wt" >/dev/null 2>&1
git -C "$SCRATCH/foo-wt" config user.email "selftest@example.com"
git -C "$SCRATCH/foo-wt" config user.name "auto-commit-selftest"
echo "nested" > "$SCRATCH/foo-wt/README"
git -C "$SCRATCH/foo-wt" add README >/dev/null 2>&1
git -C "$SCRATCH/foo-wt" commit -m "nested init" >/dev/null 2>&1
# 额外一个普通文件，确保本次提交有真实内容
echo "real" > "$SCRATCH/real.txt"
OUT="$(run_auto_commit "$SCRATCH" || true)"
assert_true "real.txt committed" git -C "$SCRATCH" cat-file -e HEAD:real.txt
assert_eq "no gitlink in HEAD for foo-wt" \
  "$(git -C "$SCRATCH" ls-tree HEAD | awk '$2 == "commit" && $4 == "foo-wt" {print $4}')" ""
assert_true "log mentions ghost gitlink removal" \
  bash -c "echo '$OUT' | grep -q '幽灵 gitlink'"
rm -rf "$SCRATCH"

# ── 4) Cursor hooks 注册 ─────────────────────────────────────────
echo "[4] Cursor project hooks register auto-commit"
HOOKS_JSON="$META_ROOT/.cursor/hooks.json"
assert_true "hooks.json exists" test -f "$HOOKS_JSON"
if [ -f "$HOOKS_JSON" ]; then
  assert_true "sessionEnd present" \
    bash -c "python3 -c \"import json; d=json.load(open('$HOOKS_JSON')); assert 'sessionEnd' in d.get('hooks',{})\""
  assert_true "stop present" \
    bash -c "python3 -c \"import json; d=json.load(open('$HOOKS_JSON')); assert 'stop' in d.get('hooks',{})\""
  assert_true "sessionEnd points to auto-commit" \
    bash -c "python3 -c \"
import json
d=json.load(open('$HOOKS_JSON'))
cmds=' '.join(h.get('command','') for h in d['hooks']['sessionEnd'])
assert 'auto-commit' in cmds
\""
  assert_true "stop uses checkpoint-threshold" \
    bash -c "python3 -c \"
import json
d=json.load(open('$HOOKS_JSON'))
cmds=' '.join(h.get('command','') for h in d['hooks']['stop'])
assert 'checkpoint-threshold' in cmds or 'auto-commit-stop' in cmds
\""
fi

echo
echo "Result: PASS=$PASS FAIL=$FAIL"
[ "$FAIL" -eq 0 ]
