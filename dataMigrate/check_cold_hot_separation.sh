#!/usr/bin/env bash
# check_cold_hot_separation.sh — CI 静态检查：冷热分离 & 分库分表合规性
#
# 检查项：
#   1. 时间累积型表的 AUTO_INCREMENT 检测 → 警告
#   2. 时间累积型表在 SQL 中无 PARTITION BY 声明 → 警告（需人工评估）
#   3. 分区运维脚本存在性检查
#
# 使用方式：
#   CI:  bash dataMigrate/check_cold_hot_separation.sh [repo_root]
#   本地: bash dataMigrate/check_cold_hot_separation.sh
#
# Exit: 0 = 无严重违规, 1 = 存在严重违规

set -euo pipefail
ROOT="${1:-$(cd "$(dirname "$0")/.." && pwd)}"

TIME_SERIES_PATTERNS="transaction|usage|event|audit|log|history|outbox|snapshot|trace|metric"

WARNING_COUNT=0
ERROR_COUNT=0

echo "=== 冷热分离 & 分库分表合规检查 ==="
echo ""

# ---- 检查 1: AUTO_INCREMENT + 分片候选表 ----
echo "--- 检查 1: AUTO_INCREMENT 使用情况 ---"
while IFS=: read -r file line content; do
    trimmed=$(echo "$content" | sed 's/^[[:space:]]*//')
    if echo "$trimmed" | grep -qE '^\s*--'; then
        continue
    fi
    echo "  ⚠️  ${file}:${line} — AUTO_INCREMENT 在分片场景下会冲突"
    echo "     建议: 使用 Snowflake / UUIDv7 全局唯一 ID"
    WARNING_COUNT=$((WARNING_COUNT + 1))
done < <(
    find "$ROOT/dataMigrate" -type f -name "*.sql" \
        -exec grep -Hn 'AUTO_INCREMENT' {} \; 2>/dev/null || true
)
echo "  AUTO_INCREMENT 警告: ${WARNING_COUNT}"
echo ""

# ---- 检查 2: 时间累积型表缺少分区声明 ----
echo "--- 检查 2: 时间累积型表 PARTITION BY 检查 ---"
while IFS=: read -r file line content; do
    trimmed=$(echo "$content" | sed 's/^[[:space:]]*//')
    if echo "$trimmed" | grep -qE '^\s*--'; then
        continue
    fi

    # grep -c exits 1 when count is 0; keep going under set -e
    has_partition=$(grep -c 'PARTITION BY' "$file" 2>/dev/null | head -1 || true)
    has_partition=${has_partition:-0}

    if [ "${has_partition}" = "0" ]; then
        if echo "$content" | grep -q 'CREATE TABLE'; then
            echo "  ℹ️  ${file}:${line} — 时间相关表未声明 PARTITION BY，请评估"
            WARNING_COUNT=$((WARNING_COUNT + 1))
        fi
    fi
done < <(
    find "$ROOT/dataMigrate" -type f -name "*.sql" \
        -exec grep -HnE "CREATE TABLE.*(${TIME_SERIES_PATTERNS})" {} \; 2>/dev/null || true
)
echo "  缺失警告: ${WARNING_COUNT}"
echo ""

# ---- 检查 3: 分区运维脚本 ----
echo "--- 检查 3: 分区运维脚本 ---"
REQUIRED_SCRIPTS=(
    "$ROOT/db/scripts/ensure_partitions.sh"
    "$ROOT/dataMigrate/check_cold_hot_separation.sh"
)

for script in "${REQUIRED_SCRIPTS[@]}"; do
    if [ -f "$script" ]; then
        echo "  ✅ ${script}"
    else
        echo "  ❌ MISSING: ${script}"
        ERROR_COUNT=$((ERROR_COUNT + 1))
    fi
done
echo ""

# ---- 检查 4: 已应用分区的表 ----
echo "--- 检查 4: 分区迁移文件 ---"
PARTITION_MIGRATIONS=(
    "$ROOT/dataMigrate/taskBill/028_native_partitioning.sql"
    "$ROOT/dataMigrate/taskCloudService/005_native_partitioning.sql"
)

for f in "${PARTITION_MIGRATIONS[@]}"; do
    if [ -f "$f" ]; then
        echo "  ✅ ${f}"
    else
        echo "  ❌ MISSING: ${f}"
        ERROR_COUNT=$((ERROR_COUNT + 1))
    fi
done

echo ""
echo "=========================================="
echo "警告: ${WARNING_COUNT}  错误: ${ERROR_COUNT}"

if [ "$ERROR_COUNT" -gt 0 ]; then
    echo "FAIL: ${ERROR_COUNT} 个错误"
    exit 1
fi

if [ "$WARNING_COUNT" -gt 0 ]; then
    echo "PASS (${WARNING_COUNT} warnings)"
else
    echo "PASS"
fi
exit 0
