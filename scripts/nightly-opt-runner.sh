#!/bin/bash
# ============================================================================
# nightly-opt-runner.sh — 夜间 OPT 自动执行器（goal-mode 无头批处理）
# ============================================================================
# crontab: */30 0-7 * * * bash /tmp/ram-work/scripts/nightly-opt-runner.sh \
#              >> /tmp/ram-work/logs/nightly-opt-runner-cron.log 2>&1
#
# 夜间每 30 分钟触发一次；若已有执行在运行（flock 占用）或不在窗口内则
# 静默退出。职责:
#   1. 窗口守卫: 仅在 00:00–08:00 之间运行（其余时间静默退出）
#   2. flock 互斥: 无运行才启动（30 分钟一次的触发源）
#   3. 调用 claude-agent run 无头执行 scripts/prompts/nightly-opt-task.md
#      （goal-mode 处理 .learnings/OPTIMIZATION_TODOS.md 中 pending 优化项）
# 自愈语义: OPT 文件本身是持久任务队列（Status 字段），会话中途死亡后
#   下一触发点自动接续剩余 pending 项，不重复不丢失。
# 会话互知: claude-agent run 内部会话经 SessionStart hook 自动注册 + Edit/Write 取锁，
#   与交互会话/夜间 sweep 冲突由 prompt 内 session check 指令协调。
# 日志: 全量输出 → logs/nightly-opt-runner-cron.log；trajectory → logs/trajectories/
# 设计: docs/superpowers/specs/2026-08-08-nightly-opt-runner-design.md
# ============================================================================
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT"

# ── 参数: --dry-run 仅打印将执行的命令 ─────────────────────────────────────
DRY_RUN=0
for arg in "$@"; do
    case "$arg" in
        --dry-run) DRY_RUN=1 ;;
        *) echo "[nightly-opt-runner] unknown arg: $arg" >&2; exit 2 ;;
    esac
done

# ── 窗口守卫（00:00 ≤ hour < 08:00；SIMULATE_HOUR / NIGHTLY_OPT_FORCE 供测试）─
HOUR="${SIMULATE_HOUR:-}"
[ -z "$HOUR" ] && HOUR="$(date +%-H 2>/dev/null || date +%H | sed 's/^0//')"
if [ "${NIGHTLY_OPT_FORCE:-0}" != "1" ] && [ "$HOUR" -ge 8 ]; then
    echo "[nightly-opt-runner] $(date '+%F %T') outside window (hour=${HOUR}); exit."
    exit 0
fi

# ── flock 互斥（无运行才启动；持锁中静默退出，避免每 30 分钟刷屏）─────────
LOCK_FILE="$ROOT/logs/.nightly-opt-runner.lock"
exec 9>"$LOCK_FILE"
if ! flock -n 9; then
    exit 0
fi

# ── API key 补齐（cron 环境不加载 ~/.bashrc → claude-agent 子进程 403）─────
if [ -z "${ANTHROPIC_API_KEY:-}" ]; then
    export ANTHROPIC_API_KEY="$(grep -m1 '^export ANTHROPIC_API_KEY=' ~/.bashrc | cut -d= -f2- | tr -d "'\"")"
fi

# ── PATH 补齐（cron 环境不加载 ~/.bashrc → 缺 .npm-global/bin 与 ~/bin）────
# bashrc 对齐: export PATH="$HOME/.npm-global/bin:$PATH"（交互 PATH 另有 ~/bin）。
# claude-agent 按 PATH 解析 claude 二进制，缺失/顺序错则找不到可用 claude。
export PATH="$HOME/.npm-global/bin:$HOME/bin:$PATH"

# ── claude 可用性预检 + 自修复（防 npm 包损坏/更新失败复发，见修复记录）───
check_claude() {
    local ver bin
    bin="$(command -v claude 2>/dev/null || true)"
    if [ -z "$bin" ]; then
        echo "[nightly-opt-runner] claude not found in PATH (searched: $PATH)"
    elif ! ver="$(timeout 30 claude --version 2>&1)"; then
        echo "[nightly-opt-runner] claude broken: $bin ($ver)"
    else
        echo "[nightly-opt-runner] claude OK: $ver ($bin)"
        return 0
    fi
    # 自修复: 跑官方 postinstall（native 依赖在位时秒修；否则提示重装）
    local pkg_dir="$HOME/.npm-global/lib/node_modules/@anthropic-ai/claude-code"
    if [ -f "$pkg_dir/install.cjs" ]; then
        echo "[nightly-opt-runner] repairing claude via install.cjs..."
        (cd "$pkg_dir" && timeout 120 node install.cjs >/dev/null 2>&1) || true
    fi
    if ! ver="$(timeout 30 claude --version 2>&1)"; then
        echo "[nightly-opt-runner] FATAL: claude unavailable after repair ($ver);"
        echo "[nightly-opt-runner]   fix manually: npm install -g @anthropic-ai/claude-code@latest --include=optional"
        exit 1
    fi
    echo "[nightly-opt-runner] claude repaired: $ver ($(command -v claude))"
}
check_claude

# 禁用自动更新：避免夜间会话触发升级时因 optional 依赖下载失败再次损坏包
#（交互会话不受影响；损坏时上方 install.cjs 自修复兜底）
export DISABLE_AUTOUPDATER=1

# deepseek-v4-flash 等第三方模型名 Claude Code 不识别时，会强制按 200k 窗口
# auto-compact，易过早压缩上下文；关闭未知模型窗口强制（见 CLI 告警文案）。
export CLAUDE_CODE_DISABLE_UNKNOWN_MODEL_WINDOW_ENFORCEMENT=1
# 显式窗口上限（token），与 DeepSeek Anthropic 兼容端点常见长上下文对齐。
export CLAUDE_CODE_MAX_CONTEXT_TOKENS="${CLAUDE_CODE_MAX_CONTEXT_TOKENS:-1000000}"

echo "[nightly-opt-runner] $(date '+%F %T') start (hour=${HOUR})"

BIN="$ROOT/claude-agent/bin/claude-agent"
# 配置优先真实文件（gitignored，本地生成），回退示例（入库，model 可能过期但可运行）
CFG="$ROOT/claude-agent/claude_config.yaml"
[ -f "$CFG" ] || CFG="$ROOT/claude-agent/claude_config.yaml.example"
PROMPT="${NIGHTLY_OPT_PROMPT:-$ROOT/scripts/prompts/nightly-opt-task.md}"
if [ ! -f "$PROMPT" ]; then
    echo "[nightly-opt-runner] prompt file not found: $PROMPT; exit."
    exit 1
fi

# trajectory 记录到 logs/trajectories/（避免在仓库内生成 .trajectories/ 弄脏工作区）
TRAJ_DIR="$ROOT/logs/trajectories"
mkdir -p "$TRAJ_DIR"
TRAJ="$TRAJ_DIR/opt-runner-$(date +%Y%m%d-%H%M%S).jsonl"

# 无头入口标记：SessionStart hook 据此注册 kind=headless-task
#（与交互会话区分，协调语义依赖该区分，见 OPT-20260808-001）。
export CLAUDE_CODE_HEADLESS=1

CMD=("$BIN" run --config-file "$CFG" --trajectory-file "$TRAJ" --file "$PROMPT")
echo "[nightly-opt-runner] cmd: ${CMD[*]}"
if [ "$DRY_RUN" = "1" ]; then
    echo "[nightly-opt-runner] dry-run; exit."
    exit 0
fi

OUT="$( "${CMD[@]}" 2>&1 )" || true
echo "$OUT"

# 提取最后一行 JSON 作为摘要（agent 完成时按 prompt 输出一行 JSON）
SUMMARY="$(printf '%s\n' "$OUT" | grep -E '^\s*\{' | tail -1 || true)"
echo "[nightly-opt-runner] $(date '+%F %T') done; trajectory=$TRAJ"
[ -n "$SUMMARY" ] && echo "[nightly-opt-runner] SUMMARY: $SUMMARY"
exit 0
