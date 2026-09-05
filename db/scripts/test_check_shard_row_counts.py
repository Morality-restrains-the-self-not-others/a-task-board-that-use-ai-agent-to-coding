#!/usr/bin/env python3
"""Self-test for check_shard_row_counts.py (no pytest required)."""

from __future__ import annotations

import importlib.util
import sys
from pathlib import Path

ROOT = Path(__file__).resolve().parents[2]
CHECKER = ROOT / "db" / "scripts" / "check_shard_row_counts.py"


def _load():
    spec = importlib.util.spec_from_file_location("check_shard_row_counts", CHECKER)
    assert spec and spec.loader
    mod = importlib.util.module_from_spec(spec)
    sys.modules[spec.name] = mod
    spec.loader.exec_module(mod)
    return mod


class FakeCursor:
    """最小游标替身：记录 execute 参数，返回脚本化结果。"""

    def __init__(self, fetchall=None, fetchone=None):
        self._fetchall = fetchall if fetchall is not None else []
        self._fetchone = fetchone if fetchone is not None else (0,)
        self.calls = []

    def execute(self, sql, params=None):
        self.calls.append((sql, params))

    def fetchall(self):
        return self._fetchall

    def fetchone(self):
        return self._fetchone


def test_shard_table_sql_has_conditions() -> None:
    mod = _load()
    sql = mod.shard_table_sql()
    assert "TABLE_SCHEMA=%s" in sql
    assert "TABLE_NAME LIKE %s" in sql
    assert "TABLE_NAME NOT LIKE %s" in sql
    assert "information_schema.tables" in sql
    assert "ORDER BY TABLE_NAME" in sql


def test_exact_count_sql_backticks_db_and_table() -> None:
    mod = _load()
    sql = mod.exact_count_sql("task_cloud", "cloud_comment_container_binding_logs_06")
    assert sql == "SELECT COUNT(*) FROM `task_cloud`.`cloud_comment_container_binding_logs_06`"


def test_fetch_rows_estimate_skips_deprecated_and_none() -> None:
    mod = _load()
    cursor = FakeCursor(fetchall=[
        ("cloud_comment_container_binding_logs_00", 0),
        ("cloud_comment_container_binding_logs_06", 15259),
        ("cloud_comment_container_binding_logs_deprecated_20260821", 1412),
    ])
    rows = mod.fetch_rows(cursor, "task_cloud", "cloud_comment_container_binding_logs_%", exact=False)
    # deprecated 由 SQL 的 NOT LIKE 参数排除，这里模拟已排除后的结果集
    assert rows == [("cloud_comment_container_binding_logs_00", 0),
                    ("cloud_comment_container_binding_logs_06", 15259)]
    # 第二个 execute 参数带 %deprecated% 排除词
    assert cursor.calls[0][1][2] == "%deprecated%"


def test_fetch_rows_exact_uses_count_star() -> None:
    mod = _load()
    # 估算阶段返回两表，exact 阶段逐表 COUNT(*) 返回固定值
    cursor = FakeCursor(fetchall=[
        ("cloud_comment_container_binding_logs_06", 15259),
        ("cloud_comment_container_binding_logs_08", 565),
    ])
    # fetchone 需要按调用次数变化：先 15259 后 565
    counter = {"n": 0}
    def _one():
        counter["n"] += 1
        return (15259,) if counter["n"] == 1 else (565,)
    cursor.fetchone = _one

    rows = mod.fetch_rows(cursor, "task_cloud", "cloud_comment_container_binding_logs_%", exact=True)
    assert rows == [("cloud_comment_container_binding_logs_06", 15259),
                    ("cloud_comment_container_binding_logs_08", 565)]
    # 首次 execute 为估算枚举，后两次为 COUNT(*) 精确
    assert cursor.calls[0][0].startswith("SELECT TABLE_NAME")
    assert cursor.calls[1][0].startswith("SELECT COUNT(*)")
    assert cursor.calls[2][0].startswith("SELECT COUNT(*)")


def test_assess_flags_over_threshold_and_totals() -> None:
    mod = _load()
    rows = [("s00", 1000), ("s06", 5_000_001), ("s08", 400)]
    res = mod.assess(rows, threshold=5_000_000, expected=16)
    assert res["over"] == [("s06", 5_000_001)]
    assert res["total"] == 5_001_401
    assert res["max_table"] == "s06"
    assert res["max_rows"] == 5_000_001
    assert res["missing"] == 13


def test_assess_empty_no_over_and_full_missing() -> None:
    mod = _load()
    res = mod.assess([], threshold=5_000_000, expected=16)
    assert res["over"] == []
    assert res["total"] == 0
    assert res["max_rows"] == 0
    assert res["max_table"] == ""
    assert res["missing"] == 16


if __name__ == "__main__":
    tests = [v for k, v in sorted(globals().items()) if k.startswith("test_")]
    for t in tests:
        t()
        print(f"PASS {t.__name__}")
    print(f"ok ({len(tests)} tests)")
