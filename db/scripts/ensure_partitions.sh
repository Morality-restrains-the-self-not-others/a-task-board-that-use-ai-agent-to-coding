#!/usr/bin/env bash
# ensure_partitions.sh — MySQL RANGE 分区自动管理
#
# 功能：
#   1. 自动检测数据库中的所有分区表
#   2. 拆分 MAXVALUE 分区，创建未来 MONTHS_AHEAD 个月的分区
#   3. 可选：删除超过 RETENTION_MONTHS 个月的旧分区（TRUNCATE PARTITION，秒级）
#
# 使用方式：
#   手动:  bash db/scripts/ensure_partitions.sh
#   Cron:  0 3 1 * * /bin/bash /path/to/db/scripts/ensure_partitions.sh >> /var/log/ensure_partitions.log 2>&1
#
# 环境变量：
#   MYSQL_HOST=127.0.0.1  MYSQL_PORT=3306  MYSQL_USER=root  MYSQL_PASSWORD=root
#   MYSQL_DATABASE=task_bill
#   MONTHS_AHEAD=3            # 提前创建未来多少个月的分区
#   RETENTION_MONTHS=12       # 保留多少个月的分区（超过则 TRUNCATE，0=不清理）
#   PARTITION_TABLES=""       # 逗号分隔的表名（为空则自动检测）
#   DRY_RUN=false

set -euo pipefail

ROOT="$(cd "$(dirname "$0")/../.." && pwd)"

MYSQL_HOST="${MYSQL_HOST:-127.0.0.1}"
MYSQL_PORT="${MYSQL_PORT:-3306}"
MYSQL_USER="${MYSQL_USER:-root}"
MYSQL_PASSWORD="${MYSQL_PASSWORD:-root}"
MYSQL_DATABASE="${MYSQL_DATABASE:-task_bill}"
MONTHS_AHEAD="${MONTHS_AHEAD:-3}"
RETENTION_MONTHS="${RETENTION_MONTHS:-12}"
PARTITION_TABLES="${PARTITION_TABLES:-}"
DRY_RUN="${DRY_RUN:-false}"

MYSQL_CMD="mysql --default-character-set=utf8mb4 -h ${MYSQL_HOST} -P ${MYSQL_PORT} -u ${MYSQL_USER} -p${MYSQL_PASSWORD} ${MYSQL_DATABASE}"

log() { echo "[$(date '+%Y-%m-%d %H:%M:%S')] $*"; }
sql_dry() {
    if [ "$DRY_RUN" = "true" ]; then
        log "[DRY RUN] $*"
    else
        $MYSQL_CMD -e "$*" 2>&1
    fi
}

# ---- 获取数据库中所有分区表 ----
get_partitioned_tables() {
    $MYSQL_CMD -N -e \
        "SELECT DISTINCT TABLE_NAME
         FROM information_schema.partitions
         WHERE TABLE_SCHEMA='${MYSQL_DATABASE}'
         AND PARTITION_NAME IS NOT NULL AND PARTITION_NAME != ''"
}

# ---- 获取表的 MAXVALUE 分区名 ----
get_maxvalue_partition() {
    local table="$1"
    $MYSQL_CMD -N -e \
        "SELECT PARTITION_NAME FROM information_schema.partitions
         WHERE TABLE_SCHEMA='${MYSQL_DATABASE}' AND TABLE_NAME='${table}'
         AND PARTITION_DESCRIPTION='MAXVALUE' LIMIT 1"
}

# ---- 获取表的分区名列表 ----
get_partition_names() {
    local table="$1"
    $MYSQL_CMD -N -e \
        "SELECT PARTITION_NAME FROM information_schema.partitions
         WHERE TABLE_SCHEMA='${MYSQL_DATABASE}' AND TABLE_NAME='${table}'
         ORDER BY PARTITION_ORDINAL_POSITION"
}

# ---- 生成未来 MONTHS_AHEAD 个月的分区定义 ----
generate_future_partitions() {
    local defs=""
    for ((i = 0; i < MONTHS_AHEAD; i++)); do
        local pname="p$(date -d "+${i} month" '+%Y%m')"
        local boundary=$(date -d "+$((i + 1)) month" '+%Y-%m-01')
        defs="${defs}PARTITION ${pname} VALUES LESS THAN (TO_DAYS('${boundary}')), "
    done
    echo "${defs%, }"
}

# ---- 确保一个表的分区就位 ----
ensure_table_partitions() {
    local table="$1"
    log "--- ${table} ---"

    local maxval_part
    maxval_part=$(get_maxvalue_partition "$table")
    if [ -z "$maxval_part" ]; then
        log "  SKIP: 无 MAXVALUE 分区"
        return 0
    fi

    local existing_names
    existing_names=$(get_partition_names "$table")

    # 检查未来 MONTHS_AHEAD 个月是否都已有分区
    local missing=false
    for ((i = 0; i < MONTHS_AHEAD; i++)); do
        local pname="p$(date -d "+${i} month" '+%Y%m')"
        if ! echo "$existing_names" | grep -q "$pname"; then
            log "  缺失: ${pname}"
            missing=true
        fi
    done

    if [ "$missing" = "false" ]; then
        log "  OK: 未来 ${MONTHS_AHEAD} 个月分区就位"
    else
        local defs
        defs=$(generate_future_partitions)
        sql_dry "ALTER TABLE ${table} REORGANIZE PARTITION ${maxval_part} INTO (
            ${defs},
            PARTITION ${maxval_part} VALUES LESS THAN MAXVALUE
        )"
        log "  DONE: 已创建未来分区"
    fi

    # 清理过期分区（如果配置了 RETENTION_MONTHS > 0）
    if [ "${RETENTION_MONTHS}" -gt 0 ]; then
        local cutoff
        cutoff=$(date -d "-${RETENTION_MONTHS} month" '+p%Y%m')
        local dropped=0
        while IFS= read -r pname; do
            # 跳过 MAXVALUE 和当前及未来分区
            if [ "$pname" = "$maxval_part" ] || [ "$pname" \> "$cutoff" ]; then
                continue
            fi
            log "  TRUNCATE 过期分区: ${pname}"
            sql_dry "ALTER TABLE ${table} TRUNCATE PARTITION ${pname}" || true
            dropped=$((dropped + 1))
        done <<< "$existing_names"
        if [ "$dropped" -gt 0 ]; then
            log "  CLEANUP: 已截断 ${dropped} 个过期分区"
        fi
    fi
}

# ---- 主流程 ----
main() {
    log "========== ensure_partitions =========="
    log "DB: ${MYSQL_DATABASE}  MONTHS_AHEAD: ${MONTHS_AHEAD}  RETENTION: ${RETENTION_MONTHS}月"
    if [ "$DRY_RUN" = "true" ]; then
        log "MODE: DRY RUN"
    fi

    local tables
    if [ -n "$PARTITION_TABLES" ]; then
        tables="$PARTITION_TABLES"
    else
        tables=$(get_partitioned_tables)
    fi

    if [ -z "$tables" ]; then
        log "未找到分区表"
        exit 0
    fi

    log "目标表: ${tables}"
    for table in $tables; do
        ensure_table_partitions "$table" || true
    done

    log "========== 完成 =========="
}

main "$@"
