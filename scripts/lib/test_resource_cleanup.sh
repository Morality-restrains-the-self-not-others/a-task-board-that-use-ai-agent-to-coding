#!/usr/bin/env bash
# ============================================================================
# test_resource_cleanup.sh — 测试临时资源清理扫描与引导 (约束 44)
# ============================================================================
# 用途: 检测测试运行遗留的临时资源并引导清理。任何测试（单测/集成/
#       Playwright/巡检）运行时临时创建 Kafka topics、MySQL/Redis 数据、
#       SQLite 文件、进程、临时文件后，均须用本脚本核验是否残留。
#
# 用法:
#   bash scripts/lib/test_resource_cleanup.sh            # 扫描（默认，只读）
#   bash scripts/lib/test_resource_cleanup.sh --fix      # 执行安全清理项
#
# 覆盖资源类型（约束 44 六类）:
#   1. Kafka junk topics（kafka-go-* 测试残留）
#   2. 孤儿测试进程（持有 db/kafka 端口连接且 ppid=1）
#   3. SQLite 残留文件（*.db / -wal / -shm）
#   4. 临时/日志大文件（logs/ 超限）
#   5. MySQL OpenTestMySQL 残留库（*_test_* / test_*；--fix 可 DROP，不含生产库）
#   6. 临时端口/容器（提示手动核验）
#
# 规则: .ai/01_project_constraints/44_test_resource_cleanup.md
# ============================================================================
set -u

ROOT="$(cd "$(dirname "$0")/../.." && pwd)"
cd "$ROOT"

FIX="${1:-}"
if [ "$FIX" = "--fix" ]; then FIX=1; else FIX=0; fi

echo "=== 测试临时资源清理扫描 ($(date '+%F %T')) ==="
echo "工作区: $ROOT"
echo

FOUND=0

# ── 1. Kafka junk topics ────────────────────────────────────────────────────
echo "── [1/6] Kafka junk topics (kafka-go-*) ──"
if python3 - <<'PY' > /tmp/trc-kafka.out 2>&1
import sys
sys.path.insert(0, "db/_infra")
from confluent_kafka.admin import AdminClient
import kafka_cleanup_junk_topics as kc
admin = AdminClient({"bootstrap.servers": kc._bootstrap_servers(kc._monorepo_root())})
junk = kc.collect_junk_topics(admin)
print(f"junk topics: {len(junk)}")
PY
then
    junk_cnt=$(grep -oP 'junk topics: \K[0-9]+' /tmp/trc-kafka.out 2>/dev/null || echo 0)
    echo "  $junk_cnt 个 kafka-go-* 残留 topics"
    if [ "${junk_cnt:-0}" -gt 0 ]; then
        FOUND=1
        echo "  清理命令: python3 db/_infra/kafka_cleanup_junk_topics.py --confirm DELETE_JUNK"
        if [ "$FIX" = "1" ]; then
            echo "  执行清理..."
            python3 db/_infra/kafka_cleanup_junk_topics.py --confirm DELETE_JUNK
        fi
    fi
else
    echo "  (检测不可用: Kafka 连接失败) $? "
    cat /tmp/trc-kafka.out | head -2
fi
rm -f /tmp/trc-kafka.out
echo

# ── 2. 孤儿测试进程（持有 db/kafka 端口连接且非 runAll 托管）───────────────
echo "── [2/6] 孤儿进程（持有 9092/9093/3306/6379 连接）──"
# 判定依据: ppid=1（runAll 以 nohup 启动）+ 持有 db/kafka 端口连接
#           + 命令行不含 runAll.yaml 中任何托管服务名（runAll 状态 API 的 pid
#             字段可能过期（进程重启后未刷新），故用服务名白名单判定）
# runAll 服务名提取（name: 字段，排除组名/注释）
MANAGED_NAMES="$(grep -E '^\s+- name: ' conf/runAll.yaml | sed 's/.*- name: //' | tr -d ' ')"
ORPHANS=""
for pid in $(ss -tnp 2>/dev/null | grep -oE 'pid=[0-9]+' | grep -oE '[0-9]+' | sort -u); do
    [ -z "$pid" ] && continue
    ppid=$(ps -o ppid= -p "$pid" 2>/dev/null | tr -d ' ')
    [ -z "$ppid" ] && continue
    [ "$ppid" != "1" ] && continue
    cmdline="$(tr '\0' ' ' < /proc/$pid/cmdline 2>/dev/null)"
    # 命令行含任一托管服务名 → runAll 托管进程，跳过
    is_managed=0
    for n in $MANAGED_NAMES; do
        case "$cmdline" in *"$n"*) is_managed=1; break ;; esac
    done
    [ "$is_managed" = "1" ] && continue
    name=$(ps -o comm= -p "$pid" 2>/dev/null | tr -d ' ')
    # 排除系统/容器宿主进程（docker、sshd 等）
    case "$name" in
        docker*|containerd*|sshd|systemd|kthreadd|promtail|node_exporter|java|code|node|nginx|mysql|redis-server) continue ;;
    esac
    if [ -n "$name" ]; then
        ORPHANS="$ORPHANS $pid($name)"
    fi
done
if [ -n "$ORPHANS" ]; then
    FOUND=1
    echo "  发现孤儿进程:$ORPHANS"
    echo "  清理命令: kill <pid> （确认非当前服务后执行）"
else
    echo "  无孤儿进程"
fi
echo

# ── 3. SQLite 残留文件 ──────────────────────────────────────────────────────
echo "── [3/6] SQLite 残留文件（*.db / -wal / -shm）──"
# 业务 SQLite（auth.db、task_cloud.db、.code-review-graph/、chrome-profile 等）
# 属正常状态文件，不报。仅报 tmp/ 下测试残留（Playwright profile、临时 DB）。
SQLITE_FILES=$(find tmp/ -type f \( -name '*.db' -o -name '*.db-wal' -o -name '*.db-shm' \) 2>/dev/null | head -20)
if [ -n "$SQLITE_FILES" ]; then
    FOUND=1
    echo "$SQLITE_FILES" | sed 's/^/  /'
    echo "  清理命令: rm -rf tmp/<测试目录>（确认是测试残留后执行）"
else
    echo "  无（业务 SQLite 状态文件不在扫描范围）"
fi
echo

# ── 4. 临时/日志大文件 ──────────────────────────────────────────────────────
echo "── [4/6] 日志/临时文件体积（>200MB）──"
BIG=$(find logs/ tmp/ -type f -size +200M 2>/dev/null | head -10)
if [ -n "$BIG" ]; then
    FOUND=1
    echo "$BIG" | while read -r f; do echo "  $(du -h "$f" | cut -f1)  $f"; done
    echo "  日志清理命令: bash scripts/truncate-ram-work-logs.sh --all"
    echo "  临时文件清理: rm -f tmp/<测试残留文件>（如 Playwright trace）"
else
    echo "  无"
fi
echo

# ── 5. MySQL 测试库残留（*_test_* / test_*；不含 registry 生产库）──────────
echo "── [5/6] MySQL 测试库残留 (OpenTestMySQL *_test_*) ──"
DROP_PY="$ROOT/db/_infra/drop_mysql_test_databases.py"
if [ -f "$DROP_PY" ]; then
    scan_out="$(python3 "$DROP_PY" --scan 2>&1)" || scan_rc=$?
    scan_rc="${scan_rc:-0}"
    echo "$scan_out" | sed 's/^/  /'
    if [ "$scan_rc" = "2" ]; then
        FOUND=1
        echo "  清理命令: python3 db/_infra/drop_mysql_test_databases.py --fix"
        if [ "$FIX" = "1" ]; then
            echo "  执行清理（不 DROP 生产库 / 默认保留 tpl_* 模板）..."
            python3 "$DROP_PY" --fix || true
        fi
    fi
else
    echo "  (脚本缺失: $DROP_PY)"
fi
echo

# ── 6. 临时端口/容器 ────────────────────────────────────────────────────────
echo "── [6/6] 临时容器（docker ps 自查项）──"
echo "  测试创建的临时容器用: docker ps | grep -vE 'kafka|redis|mysql|aimonitor|portainer|taskgateway|gitlab'"
echo "  测试监听的临时端口用: ss -tlnp | grep -vE ':80|:443|:909[23]|:3306|:6379|:9999|:4000|:18081|:18444|:801[2-9]|:802[0-5]'"
echo

# ── 汇总 ────────────────────────────────────────────────────────────────────
if [ "$FOUND" = "1" ]; then
    echo "⚠️  发现测试残留资源（见上）。请按引导命令清理。"
    exit 2
else
    echo "✅ 未发现测试残留资源。"
    exit 0
fi
