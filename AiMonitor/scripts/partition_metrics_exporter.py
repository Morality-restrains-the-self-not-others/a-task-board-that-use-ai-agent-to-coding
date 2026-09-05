#!/usr/bin/env python3
"""
partition_metrics_exporter.py — 从 MySQL information_schema 导出分区状态 Prometheus 指标

输出格式：Prometheus textfile（供 node_exporter textfile collector 采集）

指标：
  mysql_partition_rows{table, partition}     — 每个分区的行数
  mysql_partition_maxvalue_rows{table}        — p_future 分区的行数（需拆分信号）
  mysql_partition_count{table}                — 表的有效分区数（不含 p_future）
  mysql_hot_rows{table}                       — 热数据行数（最近 90 天分区之和）

使用方式：
  python3 AiMonitor/scripts/partition_metrics_exporter.py --output /var/lib/prometheus-textfiles/partition_metrics.prom

依赖：pip install pymysql
"""

import argparse
import os
import sys
from datetime import datetime, timedelta

try:
    import pymysql
except ImportError:
    print("ERROR: pymysql required. Run: pip install pymysql", file=sys.stderr)
    sys.exit(1)

# 默认口令与 docker-compose.yaml mysqld-exporter DATA_SOURCE_NAME 默认一致
# （root123456，本机开发库）；生产可用 MYSQL_TASK_BILL_PASSWORD / MYSQL_TASK_CLOUD_PASSWORD 覆盖。
DATABASES = {
    "task_bill": {"host": "127.0.0.1", "port": 3306, "user": "root", "password": "root123456"},
    "task_cloud": {"host": "127.0.0.1", "port": 3306, "user": "root", "password": "root123456"},
}

for db_name in list(DATABASES.keys()):
    prefix = f"MYSQL_{db_name.upper()}_"
    for key in ("host", "port", "user", "password"):
        env_key = f"{prefix}{key.upper()}"
        if env_key in os.environ:
            val = os.environ[env_key]
            if key == "port":
                val = int(val)
            DATABASES[db_name][key] = val

HOT_DAYS = int(os.environ.get("HOT_DAYS", "90"))


def collect_metrics() -> list[str]:
    lines = []
    lines.append("# HELP mysql_partition_rows Rows in each partition")
    lines.append("# TYPE mysql_partition_rows gauge")
    lines.append("# HELP mysql_partition_maxvalue_rows Rows in p_future MAXVALUE partition")
    lines.append("# TYPE mysql_partition_maxvalue_rows gauge")
    lines.append("# HELP mysql_partition_count Number of active (non-MAXVALUE) partitions")
    lines.append("# TYPE mysql_partition_count gauge")
    lines.append("# HELP mysql_hot_rows Estimated hot rows (partitions within HOT_DAYS)")
    lines.append("# TYPE mysql_hot_rows gauge")

    hot_cutoff = datetime.now() - timedelta(days=HOT_DAYS)

    for db_name, cfg in DATABASES.items():
        try:
            conn = pymysql.connect(
                host=cfg["host"], port=cfg["port"],
                user=cfg["user"], password=cfg["password"],
                database=db_name, charset="utf8mb4", connect_timeout=5,
            )
            cursor = conn.cursor()

            # Get all partitioned tables
            cursor.execute("""
                SELECT DISTINCT TABLE_NAME
                FROM information_schema.partitions
                WHERE TABLE_SCHEMA = %s AND PARTITION_NAME IS NOT NULL AND PARTITION_NAME != ''
            """, (db_name,))
            tables = [row[0] for row in cursor.fetchall()]

            for table in tables:
                cursor.execute("""
                    SELECT PARTITION_NAME, TABLE_ROWS, PARTITION_DESCRIPTION
                    FROM information_schema.partitions
                    WHERE TABLE_SCHEMA = %s AND TABLE_NAME = %s
                    ORDER BY PARTITION_ORDINAL_POSITION
                """, (db_name, table))

                hot_rows = 0
                active_partitions = 0

                for pname, rows, desc in cursor.fetchall():
                    label = f'{{database="{db_name}",table="{table}",partition="{pname}"}}'

                    row_count = rows if rows else 0
                    lines.append(f"mysql_partition_rows{label} {row_count}")

                    if pname == "p_future":
                        lines.append(f"mysql_partition_maxvalue_rows{{database=\"{db_name}\",table=\"{table}\"}} {row_count}")
                    else:
                        active_partitions += 1
                        # Estimate if this partition is "hot" based on partition name (pYYYYMM)
                        if pname.startswith("p") and len(pname) == 7:
                            try:
                                part_date = datetime.strptime(pname[1:], "%Y%m")
                                if part_date >= hot_cutoff:
                                    hot_rows += row_count
                            except ValueError:
                                pass

                lines.append(f"mysql_partition_count{{database=\"{db_name}\",table=\"{table}\"}} {active_partitions}")
                lines.append(f"mysql_hot_rows{{database=\"{db_name}\",table=\"{table}\"}} {hot_rows}")

            conn.close()
        except Exception as e:
            print(f"WARNING: Failed to query {db_name}: {e}", file=sys.stderr)

    lines.append(f"# EOF @ {datetime.now().isoformat()}")
    return lines


def main():
    parser = argparse.ArgumentParser(description="Export partition metrics for Prometheus")
    parser.add_argument("--output", "-o", help="Write to file instead of stdout")
    args = parser.parse_args()

    lines = collect_metrics()

    if args.output:
        tmp = args.output + ".tmp"
        with open(tmp, "w") as f:
            f.write("\n".join(lines) + "\n")
        os.rename(tmp, args.output)
        print(f"Written to {args.output} ({len(lines)} lines)", file=sys.stderr)
    else:
        print("\n".join(lines))


if __name__ == "__main__":
    main()
