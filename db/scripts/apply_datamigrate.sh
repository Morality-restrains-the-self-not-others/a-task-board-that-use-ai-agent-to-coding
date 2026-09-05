#!/usr/bin/env bash
# =============================================================================
# apply_datamigrate.sh — 统一 dataMigrate SQL 迁移执行器
# =============================================================================
# 由 runAll /api/dev/init-databases（端口 9999）和 CI 调用。
#
# 用途：对单个数据库应用 dataMigrate/<service>/*.sql 全部文件。
#       幂等 — 通过 data_migrate_log 表追踪已应用文件，重复执行无害。
#
# 用法：
#   apply_datamigrate.sh <database_name> <dataMigrate_dir> [mysql_dsn]
#
# 参数：
#   database_name  - MySQL 数据库名（如 task_auth）
#   dataMigrate_dir - dataMigrate 子目录的绝对路径（如 /repo/dataMigrate/taskAuth）
#   mysql_dsn       - 可选，MySQL 连接信息。默认从 registry.yaml 解析。
#
# 示例：
#   apply_datamigrate.sh task_auth /repo/dataMigrate/taskAuth
#   apply_datamigrate.sh task_bill /repo/dataMigrate/taskBill
#
# 行为：
#   1. 创建 data_migrate_log 表（如不存在）
#   2. 按文件名排序逐个执行未应用的 .sql 文件
#   3. 每个文件经 render_sql_file（占位符按 conf 渲染）后管道执行
#   4. 成功后记录到 data_migrate_log 实现幂等
#
# 退出码：
#   0 - 全部成功或全部已应用
#   1 - 参数错误
#   2 - MySQL 连接失败
#   3 - SQL 文件执行失败
# =============================================================================
set -euo pipefail

DB_NAME="${1:-}"
MIG_DIR="${2:-}"
MYSQL_DSN="${3:-}"

if [ -z "$DB_NAME" ] || [ -z "$MIG_DIR" ]; then
    echo "用法: $0 <database_name> <dataMigrate_dir> [mysql_dsn]" >&2
    exit 1
fi

if [ ! -d "$MIG_DIR" ]; then
    echo "[apply_datamigrate] dataMigrate 目录不存在: $MIG_DIR (跳过)" >&2
    exit 0
fi

# 解析 MySQL 连接（优先使用显式 DSN，否则从 registry.yaml 推断）
MYSQL_HOST="${MYSQL_HOST:-127.0.0.1}"
MYSQL_PORT="${MYSQL_PORT:-3306}"
MYSQL_USER="${MYSQL_USER:-taskapp}"
MYSQL_PASS="${MYSQL_PASS:-taskapp123}"

if [ -n "$MYSQL_DSN" ]; then
    # 简易 DSN 解析: user:pass@host:port
    MYSQL_USER="${MYSQL_DSN%%:*}"
    REST="${MYSQL_DSN#*:}"
    MYSQL_PASS="${REST%%@*}"
    REST="${REST#*@}"
    MYSQL_HOST="${REST%%:*}"
    MYSQL_PORT="${REST#*:}"
fi

# 查找 MySQL 容器
MYSQL_CONTAINER=""
if command -v docker >/dev/null 2>&1; then
    MYSQL_CONTAINER=$(docker compose -f "${MONOREPO_ROOT:-.}/dockerInfra/mysql/docker-compose.yml" -p docker-mysql ps -q mysql 2>/dev/null || true)
    if [ -z "$MYSQL_CONTAINER" ]; then
        MYSQL_CONTAINER=$(docker ps --filter "name=docker-mysql" --format '{{.ID}}' 2>/dev/null | head -1 || true)
    fi
fi

# mysql 执行函数
# --default-character-set=utf8mb4 确保 SQL 文件中的 UTF-8 中文不被误读为 Latin-1
mysql_exec() {
    if [ -n "$MYSQL_CONTAINER" ]; then
        docker exec -i "$MYSQL_CONTAINER" mysql --default-character-set=utf8mb4 -u "$MYSQL_USER" -p"$MYSQL_PASS" "$DB_NAME" "$@"
    else
        mysql --default-character-set=utf8mb4 -h "$MYSQL_HOST" -P "$MYSQL_PORT" -u "$MYSQL_USER" -p"$MYSQL_PASS" "$DB_NAME" "$@"
    fi
}

# 022 等文件含 __BOOTSTRAP_ADMIN_EMAIL__ 时按 conf bootstrapAdmin.email 渲染。
render_sql_file() {
    local sql_file="$1"
    if grep -Fq '__BOOTSTRAP_ADMIN_EMAIL__' "$sql_file"; then
        local helper
        helper="$(dirname "$sql_file")/bootstrap_admin_email.py"
        if [ ! -f "$helper" ]; then
            echo "[apply_datamigrate] 缺少 $helper，无法渲染 bootstrapAdmin.email" >&2
            exit 3
        fi
        python3 "$helper" render "$sql_file"
        return
    fi
    cat "$sql_file"
}

echo "[apply_datamigrate] 数据库: $DB_NAME, 目录: $MIG_DIR"

# 1. 创建追踪表（如不存在）
echo "[apply_datamigrate] 确保 data_migrate_log 表存在..."
mysql_exec -e "
CREATE TABLE IF NOT EXISTS data_migrate_log (
    step_key VARCHAR(255) PRIMARY KEY,
    applied_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    checksum VARCHAR(64) NOT NULL DEFAULT ''
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
"

# 2. 收集 SQL 文件（仅 .sql 后缀，按名称排序）
SQL_FILES=()
while IFS= read -r -d '' f; do
    SQL_FILES+=("$f")
done < <(find "$MIG_DIR" -maxdepth 1 -name '*.sql' -print0 | sort -z)

if [ ${#SQL_FILES[@]} -eq 0 ]; then
    echo "[apply_datamigrate] 无 SQL 文件，跳过"
    exit 0
fi

# 3. 逐个执行
APPLIED=0
SKIPPED=0
FAILED=0

for sql_file in "${SQL_FILES[@]}"; do
    fname="$(basename "$sql_file")"

    # 检查是否已应用
    if mysql_exec -sN -e "SELECT 1 FROM data_migrate_log WHERE step_key = '$fname'" 2>/dev/null | grep -q 1; then
        echo "[apply_datamigrate]   $fname (已应用，跳过)"
        ((SKIPPED++)) || true
        continue
    fi

    echo -n "[apply_datamigrate]   $fname ... "

    # 渲染占位符后管道执行（docker exec 无法 source 宿主机路径）
    if render_sql_file "$sql_file" | mysql_exec 2>/tmp/apply_datamigrate_err.$$; then
        mysql_exec -e "INSERT INTO data_migrate_log (step_key, applied_at) VALUES ('$fname', NOW())"
        echo "OK"
        ((APPLIED++)) || true
    else
        echo "FAILED"
        cat /tmp/apply_datamigrate_err.$$ >&2
        rm -f /tmp/apply_datamigrate_err.$$
        ((FAILED++)) || true
        echo "[apply_datamigrate] 错误: $fname 执行失败 (数据库: $DB_NAME)" >&2
        exit 3
    fi
    rm -f /tmp/apply_datamigrate_err.$$
done

echo "[apply_datamigrate] 完成: $APPLIED 已应用, $SKIPPED 已跳过, $FAILED 失败 (数据库: $DB_NAME)"
exit 0
