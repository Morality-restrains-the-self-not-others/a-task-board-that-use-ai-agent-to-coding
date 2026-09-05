#!/usr/bin/env bash
# ============================================================================
# auto-commit.sh — 会话结束自动提交（SessionEnd hook）+ 会话中途检查点（Stop hook）— 项目元规则
# ============================================================================
# 用途: 每个 Claude Code / Cursor 会话结束时，将 meta 仓库工作区全部变更
#       自动提交到 main（本仓库主工作流），作为跨会话误删/丢失的恢复安全网。
#       Stop / stop hook 可带 --checkpoint-threshold 复用本脚本做会话中途低频
#       检查点，覆盖终端强杀/断电/崩溃等 SessionEnd 无法触发的异常退出场景。
#
# 设计决策（与既有门禁共存 — 第 6 条「禁止 --no-verify」为核心规则，绝不绕过）:
#   1. 走真实 pre-commit 门禁（.githooks v1.1.0）: 会话锁校验（同步执行时
#      提交进程祖先链含会话注册 pid → 自我识别放行）、子仓优先提交门禁、
#      随机单测抽测、commit-msg bug-fix 门禁。门禁未通过 → 提交失败，工作区
#      保持原样（git 不丢未提交内容），输出原因由用户人工处理。
#   2. 子仓指针: 暂存后剔除「脏 WIP 子仓」的 gitlink 变更（遵循
#      .ai/01_project_constraints/32_submodule_commit_order.md 既有约定：
#      只同步 clean 子仓指针）。
#   3. 幂等: 无变更 → no-op；merge/rebase/cherry-pick 中间状态 → 跳过；
#      .env* 敏感文件在工作区 → 跳过（防密钥入库）。
#   4. 并行会话竞争: index.lock 冲突重试 3 次。
#   5. 服务登记: 每次触发先运行 register-precise-restart-scan.sh，将本轮源码
#      变更的 runAll 托管服务自动登记到 .runall/precise_restart_services.txt
#      （42 号约束文强制机制；仅文档/测试变更不触发，fail-open）。
#   5b. trae-agent 镜像推送: 每次触发运行 trae-agent-docker-push-scan.sh 登记
#      pending（46 号约束）；SessionEnd（非 checkpoint）若仍 pending 则以后台
#      兜底执行 DOCKER_PUSH=1 ./buildDocker.sh（fail-open，不阻断提交）。
#   5c. 残留 worktree 扫描: cleanup_stale_worktrees.py --scan 写入
#      .runall/stale_worktrees.txt（21 号约束；只报告不拆除，fail-open）。
#   6. 落点强制 main: 非 main 时先 switch/checkout 到 main 再提交；detached
#      HEAD 或无法切换则跳过（避免提交落到 feature 分支）。
#
# SSOT: CLAUDE.md（元规则文档）、scripts/hooks/templates/claude-settings.json
#       （Claude hook 模板）、.cursor/hooks.json（Cursor sessionEnd/stop）。
# 注意: 分发到子仓后本脚本仍操作 meta 仓库（@META_ROOT@ 恒为 meta 根）。
# 测试: AUTO_COMMIT_REPO_ROOT 可覆盖目标仓库（仅自测）；见 auto-commit_selftest.sh。
# ============================================================================
set -u

# ── 参数解析 ──────────────────────────────────────────────────────
# --checkpoint-threshold <秒>: Stop hook 低频检查点模式（见下方阈值逻辑）；
#   不带参数时为 SessionEnd 全量自动提交（原行为不变）。
CHECKPOINT_THRESHOLD=""
while [ $# -gt 0 ]; do
  case "$1" in
    --checkpoint-threshold)
      CHECKPOINT_THRESHOLD="${2:-}"
      shift 2 || break
      ;;
    *)
      shift
      ;;
  esac
done

# ── 定位 meta 仓库根（脚本位于 <meta-root>/scripts/lib/）──────────
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
# AUTO_COMMIT_REPO_ROOT: 自测覆盖；生产路径恒为脚本所在 meta 根
if [ -n "${AUTO_COMMIT_REPO_ROOT:-}" ]; then
  REPO_ROOT="$AUTO_COMMIT_REPO_ROOT"
else
  REPO_ROOT="$(git -C "$SCRIPT_DIR" rev-parse --show-toplevel 2>/dev/null)" || exit 0
fi
cd "$REPO_ROOT" || exit 0
TARGET_BRANCH="${AUTO_COMMIT_TARGET_BRANCH:-main}"

# ── 中间状态跳过（merge/rebase/cherry-pick/bisect）────────────────
for ref in MERGE_HEAD REBASE_HEAD CHERRY_PICK_HEAD BISECT_LOG; do
  if git rev-parse -q --verify "$ref" >/dev/null 2>&1; then
    echo "auto-commit skipped: $ref exists（merge/rebase 进行中）" >&2
    exit 0
  fi
done
GIT_DIR="$(git rev-parse --git-dir 2>/dev/null || true)"
if [ -n "$GIT_DIR" ] && { [ -d "$GIT_DIR/rebase-merge" ] || [ -d "$GIT_DIR/rebase-apply" ]; }; then
  echo "auto-commit skipped: rebase in progress" >&2
  exit 0
fi

# ── 服务源码变更 → 精准编译重启自动登记（42 号约束文强制机制）────────
# 每次 Stop/SessionEnd 扫描脏子模块，将源码/构建/配置有变更的 runAll 托管服务
# 自动登记到 .runall/precise_restart_services.txt。登记与提交解耦：
# 不受 checkpoint 阈值与提交门禁影响；扫描失败不阻断提交（fail-open）。
bash "$SCRIPT_DIR/register-precise-restart-scan.sh" || true

# ── clone-run 产物归集（gitignored deploy-binaries/；无 ELF 源则跳过）──
bash "$SCRIPT_DIR/collect-deploy-binaries-on-commit.sh" || true

# ── trae-agent 镜像相关变更 → Docker 推送 pending 登记（46 号约束）──
bash "$SCRIPT_DIR/trae-agent-docker-push-scan.sh" || true

# ── 残留 *-wt / 额外 worktree 扫描（21 号约束；只写报告，不自动拆除）──
if [ -f "$REPO_ROOT/runAll/scripts/cleanup_stale_worktrees.py" ]; then
  python3 "$REPO_ROOT/runAll/scripts/cleanup_stale_worktrees.py" --scan --root "$REPO_ROOT" >/dev/null 2>&1 || true
fi

# SessionEnd（无 checkpoint 阈值）退出时若仍 pending，后台兜底推送（不阻塞 hook）。
_trae_docker_push_on_session_end() {
  if [ -z "${CHECKPOINT_THRESHOLD:-}" ]; then
    bash "$SCRIPT_DIR/trae-agent-docker-push.sh" --if-pending --background || true
  fi
}
trap '_trae_docker_push_on_session_end' EXIT

# ── checkpoint 阈值去重（Stop hook 每轮触发，仅按阈值低频提交）────
# 状态文件记录上次成功检查点提交的 unix 时间戳；距上次未达阈值 → 直接退出。
# 幂等（同内容 no-op）由下方 git diff 空检查兜底，内容哈希去重由 git 自身
# 完成（工作区与 HEAD 一致即无提交），故阈值语义为「距上次成功提交 N 秒内
# 不重复提交」。状态文件放 git 目录（.git/），不入版本库、不污染历史。
GIT_DIR_META="$(git rev-parse --absolute-git-dir 2>/dev/null || echo "$REPO_ROOT/.git")"
CHECKPOINT_STATE_FILE="$GIT_DIR_META/auto-commit-checkpoint"
if [ -n "$CHECKPOINT_THRESHOLD" ]; then
  if ! [[ "$CHECKPOINT_THRESHOLD" =~ ^[0-9]+$ ]] || [ "$CHECKPOINT_THRESHOLD" -le 0 ]; then
    echo "auto-commit: 非法 checkpoint-threshold: $CHECKPOINT_THRESHOLD（须为正整数秒）" >&2
    exit 0
  fi
  now="$(date +%s)"
  last=0
  if [ -f "$CHECKPOINT_STATE_FILE" ]; then
    last="$(cat "$CHECKPOINT_STATE_FILE" 2>/dev/null || echo 0)"
    case "$last" in
      ''|*[!0-9]*) last=0 ;;
    esac
  fi
  if [ $((now - last)) -lt "$CHECKPOINT_THRESHOLD" ]; then
    exit 0
  fi
  CHECKPOINT_MODE=1
fi

# 仅在检查点模式且发生实际提交/确定无内容可提交时推进状态
write_checkpoint_state() {
  [ "${CHECKPOINT_MODE:-}" = "1" ] || return 0
  echo "$(date +%s)" > "$CHECKPOINT_STATE_FILE" 2>/dev/null || true
}

# ── 无变更即 no-op（幂等）─────────────────────────────────────────
if git diff --quiet && git diff --cached --quiet &&
   [ -z "$(git ls-files --others --exclude-standard)" ]; then
  exit 0
fi

# ── 强制落点 main（本仓库主工作流）────────────────────────────────
# 有可提交变更时才切换，避免干净工作区无谓改分支。detached HEAD 或无法
# 切换到目标分支时跳过，防止自动提交落到 feature 分支。
current_branch="$(git branch --show-current 2>/dev/null || true)"
if [ -z "$current_branch" ]; then
  echo "auto-commit skipped: detached HEAD，无法保证提交到 ${TARGET_BRANCH}" >&2
  exit 0
fi
if [ "$current_branch" != "$TARGET_BRANCH" ]; then
  echo "auto-commit: 当前分支为 ${current_branch}，切换到 ${TARGET_BRANCH} 后提交" >&2
  if ! git switch "$TARGET_BRANCH" >/dev/null 2>&1 &&
     ! git checkout "$TARGET_BRANCH" >/dev/null 2>&1; then
    echo "auto-commit skipped: 无法切换到 ${TARGET_BRANCH}（冲突或分支不存在）— 请在 ${TARGET_BRANCH} 上工作" >&2
    exit 0
  fi
fi

# ── 敏感文件守卫（.env* 禁止入库）─────────────────────────────────
# 用 name-only 输出（无 `XY ` 前缀），路径以 .env 开头的段即命中
if { git ls-files --others --exclude-standard; git diff --name-only; git diff --cached --name-only; } 2>/dev/null | grep -Eq '(^|/)(\.env([./]|$))'; then
  echo "auto-commit skipped: 工作区含 .env* 敏感文件 — 请人工处理后提交" >&2
  exit 0
fi

# ── index.lock 竞争退避重试（并行会话提交防护）───────────────────
git_retry() {
  local i log
  for i in 1 2 3; do
    log="$(mktemp /tmp/auto-commit.XXXXXX.log)"
    if "$@" >"$log" 2>&1; then
      rm -f "$log"
      return 0
    fi
    # index.lock 竞争 + ref CAS 竞态（门禁期间他会话提交推进 HEAD）均重试
    if grep -Eq 'index\.lock|cannot lock ref' "$log"; then
      rm -f "$log"
      [ $i -lt 3 ] && sleep 2
      continue
    fi
    cat "$log" >&2
    rm -f "$log"
    return 1
  done
  echo "auto-commit: index.lock 持续被占用（其他会话提交进行中）— 跳过，下个会话结束重试" >&2
  return 1
}

# ── 暂存全部（含删除；尊重 .gitignore）────────────────────────────
git_retry git add -A || exit 0

# ── 剔除脏 WIP 子仓的 gitlink 变更（32 号专文约定）────────────────
while IFS= read -r sub; do
  [ -z "$sub" ] && continue
  if [ -d "$REPO_ROOT/$sub/.git" ] || [ -f "$REPO_ROOT/$sub/.git" ]; then
    if [ -n "$(git -C "$REPO_ROOT/$sub" status --porcelain 2>/dev/null)" ]; then
      git restore --staged -- "$sub" 2>/dev/null || git reset -q -- "$sub" 2>/dev/null
      echo "auto-commit: 剔除脏子仓 $sub 的 gitlink 指针变更" >&2
    fi
  fi
done < <(git config --file "$REPO_ROOT/.gitmodules" --get-regexp '^submodule\..*\.path$' 2>/dev/null | awk '{print $2}')

# ── 剔除未登记于 .gitmodules 的幽灵 gitlink（OPT-20260815-007）────
# 兜底：worktree/嵌套 git 目录若因 .gitignore 遗漏被 git add -A 收成
# 160000 模式，这里按 .gitmodules 白名单反选剔除，防止幽灵指针进历史。
while IFS= read -r line; do
  [ -z "$line" ] && continue
  # line 形如 "160000 <sha> 0\t<path>" — 用 tab 后段作为路径
  path="${line#*$'\t'}"
  if ! git config --file "$REPO_ROOT/.gitmodules" --get "submodule.$path.path" >/dev/null 2>&1; then
    git restore --staged -- "$path" 2>/dev/null || git reset -q -- "$path" 2>/dev/null
    echo "auto-commit: 剔除未登记幽灵 gitlink $path（不在 .gitmodules）" >&2
  fi
done < <(git ls-files -s | awk '$1 == "160000"')

# ── 最终确认仍有变更（可能只剩被剔除的指针）───────────────────────
if git diff --cached --quiet; then
  # 变更全部为脏子仓指针（已剔除）→ 无内容可提交；推进 checkpoint 状态防重复扫描
  write_checkpoint_state
  exit 0
fi

# ── 提交（走完整门禁；禁止 --no-verify 核心规则）──────────────────
TS="$(date '+%Y-%m-%d %H:%M:%S')"
N="$(git diff --cached --name-only | wc -l)"
if [ "${CHECKPOINT_MODE:-}" = "1" ]; then
  MSG="chore(auto-commit): 会话中途检查点提交（${TS}，${N} 文件变更）

会话中途由 Stop hook 低频检查点自动提交（项目元规则，见 CLAUDE.md），
作为终端强杀/断电/崩溃等异常退出场景的恢复安全网。本提交走完整
pre-commit/commit-msg 门禁；脏 WIP 子仓的 gitlink 变更已剔除（32 号专文约定）。"
else
  MSG="chore(auto-commit): 会话结束自动提交检查点（${TS}，${N} 文件变更）

会话结束时由 SessionEnd hook 自动提交（项目元规则，见 CLAUDE.md），
作为跨会话误删/丢失的恢复安全网。本提交走完整 pre-commit/commit-msg
门禁；脏 WIP 子仓的 gitlink 变更已剔除（32 号专文约定）。"
fi

if git_retry git commit -m "$MSG"; then
  echo "auto-commit: $(git rev-parse --short HEAD) — ${N} 个文件已提交（门禁通过）" >&2
  write_checkpoint_state
else
  # OPT-20260810-019: 区分「并行进程已提交」与「真门禁失败」。
  # 并发 Stop/SessionEnd 竞态时，另一进程可能已提交同一批变更并使 HEAD 推进，
  # 本进程 commit 失败（nothing to commit）但工作区已干净 → 视为成功，非误报。
  if [ -z "$(git status --porcelain)" ]; then
    echo "auto-commit: 变更已被其他会话提交（HEAD=$(git rev-parse --short HEAD)）— 视为成功" >&2
    write_checkpoint_state
  else
    echo "auto-commit FAILED: 提交门禁未通过或提交失败 — 工作区保持原样（内容未丢失），请按上方输出人工处理" >&2
  fi
fi
exit 0
