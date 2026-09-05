#!/usr/bin/env python3
"""Unit tests for check_mysql_compat.py — MySQL compatibility checker."""
from __future__ import annotations

from pathlib import Path

import check_mysql_compat as chk


def _issues(content: str) -> list[dict]:
    return chk.check_sql_text(content, Path("taskBill/038_order_comments.sql"))


def test_datetime_column_type_not_flagged():
    """MySQL 合法列类型 DATETIME(fsp) 不应被误判为 SQLite datetime() 函数（OPT-20260811-080 顺带）。"""
    sql = """CREATE TABLE IF NOT EXISTS billing_order_comment (
  id BIGINT NOT NULL PRIMARY KEY,
  created_at DATETIME(6) NOT NULL,
  INDEX idx (order_id, created_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
"""
    issues = _issues(sql)
    assert not issues, issues


def test_datetime_function_still_flagged():
    """SQLite datetime() 函数调用仍须被拦截。"""
    sql = "UPDATE t SET updated_at = datetime('now');"
    issues = _issues(sql)
    assert any('datetime()' in i['message'] for i in issues), issues


def test_datetime_column_arg_still_flagged():
    """非小整数参数的 datetime(...) 调用（列引用）仍须被拦截。"""
    sql = "SELECT datetime(created_at) FROM t;"
    issues = _issues(sql)
    assert any('datetime()' in i['message'] for i in issues), issues


def test_strftime_still_flagged():
    sql = "SELECT strftime('%Y', created_at) FROM t;"
    issues = _issues(sql)
    assert any('strftime()' in i['message'] for i in issues), issues
