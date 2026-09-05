#!/usr/bin/env python3
"""Audit billing.transaction task_id formats vs taskTaskService DB.

Reports:
  - canonical ``task_*`` rows
  - legacy bare snowflake digits (still resolvable / orphaned)
  - other / empty

Usage:
  python3 taskBill/scripts/audit_billing_task_ids.py --billing-db PATH --task-db PATH

DEPRECATED (2026-08-24, docs-cleanup): 存储已全部 MySQL 化（db/registry.yaml
driver: mysql；task_bill / task_task 为 MySQL 库），SQLite 默认路径已失效。
如需运行，请传入 SQLite 快照文件（如 mysqldump 导出后重建），或改用
mysql 客户端直接查询 MySQL 库。
"""

from __future__ import annotations

import argparse
import sqlite3
import sys
from pathlib import Path


def main() -> int:
    ap = argparse.ArgumentParser(description=__doc__)
    ap.add_argument("--billing-db", type=Path, default=None)
    ap.add_argument("--task-db", type=Path, default=None)
    args = ap.parse_args()

    if not args.billing_db or not args.task_db:
        print(
            "存储已 MySQL 化：不再有默认 SQLite 路径，必须显式传 --billing-db / --task-db"
            "（SQLite 快照文件）。详见脚本 docstring。",
            file=sys.stderr,
        )
        return 2

    if not args.billing_db.is_file():
        print(f"billing db missing: {args.billing_db}", file=sys.stderr)
        return 2
    if not args.task_db.is_file():
        print(f"task db missing: {args.task_db}", file=sys.stderr)
        return 2

    bill = sqlite3.connect(f"file:{args.billing_db}?mode=ro", uri=True)
    task = sqlite3.connect(f"file:{args.task_db}?mode=ro", uri=True)
    try:
        rows = bill.execute(
            """
            SELECT DISTINCT task_id FROM billing_transaction
            WHERE task_id IS NOT NULL AND TRIM(task_id) != ''
            """
        ).fetchall()
        task_ids = {r[0] for r in task.execute("SELECT id FROM tasks").fetchall()}

        canonical = []
        legacy_ok = []
        orphan = []
        other = []
        for (raw,) in rows:
            tid = str(raw).strip()
            if tid.startswith("task_"):
                canonical.append(tid)
                continue
            if tid.isdigit():
                (legacy_ok if tid in task_ids else orphan).append(tid)
                continue
            other.append(tid)

        print(f"distinct task_id values: {len(rows)}")
        print(f"  canonical task_*:     {len(canonical)}")
        print(f"  legacy bare (alive):  {len(legacy_ok)}")
        print(f"  legacy bare (orphan): {len(orphan)}")
        print(f"  other:                {len(other)}")
        if orphan:
            print("orphan samples:")
            for tid in orphan[:20]:
                print(f"  - {tid}")
        if other:
            print("other samples:")
            for tid in other[:20]:
                print(f"  - {tid}")
        return 0
    finally:
        bill.close()
        task.close()


if __name__ == "__main__":
    raise SystemExit(main())
