#!/usr/bin/env python3
"""check_shard_row_counts.py — 哈希分片表行数监控（OPT-20260820-024）。

task_cloud 的云资源日志按 workspace_id 哈希分 16 片（ccbLogShardCount /
jobExecutionEventShardCount，见 taskCloudService/src/ccb_log_shard.go 与
job_execution_event_shard.go）。单片热数据涨到百万级后，运维裁冷数据 / 改 DDL
会锁整片；本脚本用于提前发现热片，避免在热片上重演锁表迁移。

监控对象：非 RANGE 分区的哈希分片表族（partition_metrics_exporter.py 只覆盖
information_schema.partitions 里的 RANGE 分区，不覆盖独立的分片表）。
数据源：information_schema.tables.TABLE_ROWS（InnoDB 估算，秒级）。

行为：
  - 对每个分片表族输出每片行数分布；
  - 任一分片（非 deprecated 表）超过 THRESHOLD_ROWS 时以退出码 1 结束（便于 cron 告警）；
  - 分片表数量少于 EXPECTED_SHARD_COUNT 时打印 WARN（表缺失会让该片写入静默失败）。

使用方式：
  python3 db/scripts/check_shard_row_counts.py
  python3 db/scripts/check_shard_row_counts.py --database task_cloud --threshold 5000000
  python3 db/scripts/check_shard_row_counts.py --exact          # 精确 COUNT(*)，重

环境变量（与 AiMonitor/scripts/partition_metrics_exporter.py 对齐）：
  MYSQL_<DB大写>_HOST/PORT/USER/PASSWORD   连接配置（默认 127.0.0.1:3306 root/root）
  THRESHOLD_ROWS=5000000                   单片行数告警阈值
  EXPECTED_SHARD_COUNT=16                  每族期望分片数（改 DDL 时同步 Go 常量）
  EXACT_COUNT=false                        true 时用 SELECT COUNT(*) 精确统计
"""
from __future__ import annotations

import argparse
import os
import sys
from typing import Any

try:
    import pymysql
except ImportError:
    print("ERROR: pymysql required. Run: pip install pymysql", file=sys.stderr)
    sys.exit(1)

# 哈希分片表族 → 期望片数。改 DDL 时同步 taskCloudService/src/*_shard.go 的常量。
FAMILIES: dict[str, int] = {
    "cloud_comment_container_binding_logs_%": 16,
    "cloud_job_execution_event_%": 16,
}

# 默认监控数据库（可用 --database 覆盖；连接细节由 MYSQL_<DB>_* 环境变量覆盖）。
DATABASES: dict[str, dict[str, Any]] = {
    "task_cloud": {"host": "127.0.0.1", "port": 3306, "user": "root", "password": "root"},
}


def apply_env(cfg: dict[str, Any], db_name: str) -> None:
    prefix = f"MYSQL_{db_name.upper()}_"
    for key in ("host", "port", "user", "password"):
        env_key = prefix + key.upper()
        if env_key in os.environ:
            val: Any = os.environ[env_key]
            cfg[key] = int(val) if key == "port" else val


def shard_table_sql() -> str:
    """返回按分片表族枚举每片 TABLE_ROWS 的 SQL（估算，秒级）。"""
    return (
        "SELECT TABLE_NAME, TABLE_ROWS FROM information_schema.tables "
        "WHERE TABLE_SCHEMA=%s AND TABLE_NAME LIKE %s "
        "AND TABLE_NAME NOT LIKE %s ORDER BY TABLE_NAME"
    )


def exact_count_sql(db_name: str, table: str) -> str:
    """返回精确行数的 SQL。表名来自 information_schema 且匹配 *_shard 命名，可安全反引号引用。"""
    return f"SELECT COUNT(*) FROM `{db_name}`.`{table}`"


def fetch_rows(cursor: Any, db_name: str, pattern: str, exact: bool) -> list[tuple[str, int]]:
    """返回 (表名, 行数) 列表；exact=True 时逐表 COUNT(*)。"""
    cursor.execute(shard_table_sql(), (db_name, pattern, "%deprecated%"))
    # SQL 层已排除 deprecated；Python 侧再过滤一次作纵深防御（自定义 pattern 匹配到
    # deprecated 表时不会把已停写表误当热片）。
    found = [(row[0], int(row[1] or 0)) for row in cursor.fetchall() if "deprecated" not in row[0]]
    if not exact:
        return found
    precise: list[tuple[str, int]] = []
    for table, _ in found:
        cursor.execute(exact_count_sql(db_name, table))
        precise.append((table, int(cursor.fetchone()[0])))
    return precise


def assess(
    rows_by_table: list[tuple[str, int]], threshold: int, expected: int
) -> dict[str, Any]:
    """评估分片行数分布：返回热片、最大片、总量与缺失族警告。"""
    over = [(t, r) for t, r in rows_by_table if r > threshold]
    total = sum(r for _, r in rows_by_table)
    max_rows = max((r for _, r in rows_by_table), default=0)
    max_table = max(rows_by_table, key=lambda p: p[1], default=("", 0))[0]
    missing = max(0, expected - len(rows_by_table))
    return {
        "over": over,
        "total": total,
        "max_rows": max_rows,
        "max_table": max_table,
        "missing": missing,
    }


def run(db_name: str, threshold: int, expected: int, exact: bool) -> int:
    cfg = dict(DATABASES.get(db_name, {"host": "127.0.0.1", "port": 3306, "user": "root", "password": "root"}))
    apply_env(cfg, db_name)
    conn = pymysql.connect(
        host=cfg["host"], port=cfg["port"], user=cfg["user"], password=cfg["password"],
        database=db_name, charset="utf8mb4", connect_timeout=5,
    )
    exit_code = 0
    try:
        with conn.cursor() as cursor:
            for pattern, expected_count in FAMILIES.items():
                rows = fetch_rows(cursor, db_name, pattern, exact)
                res = assess(rows, threshold, expected_count)
                family = pattern.rstrip("%")
                print(f"[{db_name}] 分片族 {family}* : {len(rows)} 片, 总行数 {res['total']}, "
                      f"最大片 {res['max_table']}={res['max_rows']}")
                for table, r in rows:
                    mark = "  <-- OVER_THRESHOLD" if r > threshold else ""
                    print(f"    {table}  {r}{mark}")
                for table, r in res["over"]:
                    print(f"    WARN: {table} 行数 {r} 超过阈值 {threshold}", file=sys.stderr)
                    exit_code = 1
                if res["missing"] > 0:
                    print(f"    WARN: {family}* 仅 {len(rows)} 片, 期望 {expected_count} 片"
                          f"（缺失 {res['missing']} 片, 该片写入会失败）", file=sys.stderr)
    finally:
        conn.close()
    return exit_code


def main(argv: list[str] | None = None) -> int:
    ap = argparse.ArgumentParser(description=__doc__)
    ap.add_argument("--database", default="task_cloud", help="监控数据库名（默认 task_cloud）")
    ap.add_argument("--threshold", type=int, default=int(os.environ.get("THRESHOLD_ROWS", "5000000")),
                    help="单片行数告警阈值（默认 5000000）")
    ap.add_argument("--expected-shards", type=int,
                    default=int(os.environ.get("EXPECTED_SHARD_COUNT", "16")),
                    help="每族期望分片数（默认 16）")
    ap.add_argument("--exact", action="store_true",
                    default=os.environ.get("EXACT_COUNT", "").lower() in ("1", "true", "yes"),
                    help="精确 COUNT(*) 统计（默认估算 TABLE_ROWS）")
    args = ap.parse_args(argv)
    try:
        return run(args.database, args.threshold, args.expected_shards, args.exact)
    except Exception as e:
        print(f"ERROR: 查询 {args.database} 失败: {e}", file=sys.stderr)
        return 2


if __name__ == "__main__":
    sys.exit(main())
