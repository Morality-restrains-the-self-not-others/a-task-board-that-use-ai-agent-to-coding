#!/bin/bash
# ============================================================================
# nightly-test-sweep.sh — 夜间随机单测巡检与自愈入口 (架构 v65)
# ============================================================================
# crontab: */20 0-7 * * * bash /tmp/ram-work/scripts/nightly-test-sweep.sh \
#              >> /tmp/ram-work/logs/nightly-test-sweep-cron.log 2>&1
#
# 夜间每 20 分钟触发一次；若已有巡检在运行（flock 占用）则静默退出，
# 否则启动一轮新巡检（编排器内部最多 3 轮 / 到 07:00 截止）。
# 职责:
#   1. 窗口守卫: 仅在 00:00–08:00 之间运行（其余时间静默退出）
#   2. flock 互斥: 无运行才启动（20 分钟一次的触发源）
#   3. 调用编排器 scripts/nightly_test_sweep.py
# 规则: .ai/01_project_constraints/28_commit_random_unit_test_debt_fix.md
# 设计: docs/superpowers/specs/2026-08-05-nightly-test-sweep-design.md
# ============================================================================
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT"

# ── 窗口守卫（00:00 ≤ hour < 08:00）────────────────────────────────────────
HOUR="$(date +%-H 2>/dev/null || date +%H | sed 's/^0//')"
if [ "$HOUR" -ge 8 ]; then
    echo "[nightly-test-sweep] $(date '+%F %T') outside window (hour=${HOUR}); exit."
    exit 0
fi

# ── flock 互斥（无运行才启动；持锁中静默退出，避免每 20 分钟刷屏）─────────
LOCK_FILE="$ROOT/logs/.nightly-test-sweep.lock"
exec 9>"$LOCK_FILE"
if ! flock -n 9; then
    exit 0
fi

echo "[nightly-test-sweep] $(date '+%F %T') start (hour=${HOUR})"
# 透传参数（--repo / --ratio / --no-fix / --dry-run / --exclude / --simulate-hour / --max-rounds / --round-cutoff）
exec python3 scripts/nightly_test_sweep.py "$@"
