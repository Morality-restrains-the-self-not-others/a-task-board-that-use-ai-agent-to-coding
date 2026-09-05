#!/usr/bin/env bash
# ============================================================================
# sessionctl — claude-agent session CLI 薄封装（agent-session-coordination v14/v66）
# ============================================================================
# 供 Claude Code hooks / nightly sweep / pre-commit / 手工使用。逻辑 SSOT 为
# claude-agent Go 二进制的 `session` 子命令；本脚本仅负责定位与构建二进制。
#
# 用法: bash scripts/lib/sessionctl.sh <session 子命令与参数...>
#   示例: bash scripts/lib/sessionctl.sh register --kind interactive --pid $CLAUDE_PROCESS_ID
#         bash scripts/lib/sessionctl.sh precheck --pid $CLAUDE_PROCESS_ID
#         bash scripts/lib/sessionctl.sh release --all --pid $CLAUDE_PROCESS_ID
#
# 环境: CLAUDE_AGENT_BIN 可覆盖二进制路径；SESSION_HUB_DIR 可覆盖 meta root（默认从 cwd 向上发现 .gitmodules）。
# 设计: docs/superpowers/specs/2026-08-06-agent-session-coordination-design.md
# ============================================================================
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
META_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"

BIN="${CLAUDE_AGENT_BIN:-$META_ROOT/claude-agent/bin/claude-agent}"
if [ ! -x "$BIN" ]; then
    # 未构建则按需构建（幂等；构建失败不阻塞 hook — 由调用方决定）
    (cd "$META_ROOT/claude-agent" && ./build.sh >/dev/null 2>&1 || true)
fi
if [ ! -x "$BIN" ]; then
    echo "sessionctl: claude-agent binary not found: $BIN (构建失败?)" >&2
    exit 0
fi

exec "$BIN" session "$@"
