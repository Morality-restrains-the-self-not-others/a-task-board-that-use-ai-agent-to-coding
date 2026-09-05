#!/usr/bin/env python3
"""Self-test for archive_impersonation_sessions.py (no pytest required; follows
test_check_shard_row_counts.py importlib pattern)."""

from __future__ import annotations

import importlib.util
import sys
from pathlib import Path

ROOT = Path(__file__).resolve().parents[2]
SCRIPT = ROOT / "db" / "scripts" / "archive_impersonation_sessions.py"


def _load():
    spec = importlib.util.spec_from_file_location("archive_impersonation_sessions", SCRIPT)
    assert spec and spec.loader
    mod = importlib.util.module_from_spec(spec)
    sys.modules[spec.name] = mod
    spec.loader.exec_module(mod)
    return mod


class FakeCursor:
    """最小游标替身：会话查询返回 3 个 id、孤儿收信箱返回 2 个 id，随后为空。"""

    def __init__(self):
        self.calls = []
        self._phase = 0

    def execute(self, sql, params=None):
        self.calls.append((sql, params))
        self._sql = sql
        self._params = params or ()

    def fetchall(self):
        if "FROM auth_impersonation_session" in self._sql and "archive" not in self._sql:
            if self._phase == 0:
                self._phase = 1
                return [(11,), (12,), (13,)]
            return []
        if "FROM auth_user_inbox_message" in self._sql and "archive" not in self._sql:
            if self._phase == 1:
                self._phase = 2
                return [(21,), (22,)]
            return []
        return []


class FakeConn:
    def __init__(self):
        self.commits = 0

    def cursor(self):
        return FakeCursor()

    def commit(self):
        self.commits += 1

    def rollback(self):
        pass


def test_sql_generation() -> None:
    mod = _load()
    ssql, sp = mod.select_session_ids_sql(90, 1000)
    assert ssql.count("%s") == 3
    assert sp == (90, 90, 1000)
    assert "ended_at < (UTC_TIMESTAMP() - INTERVAL %s DAY)" in ssql
    assert "expires_at < (UTC_TIMESTAMP() - INTERVAL %s DAY)" in ssql
    isql, ip = mod.select_orphan_inbox_ids_sql(90, 1000)
    assert ip == (90, 1000)
    assert "impersonation_session_id IS NULL" in isql


def test_run_batches_and_commit() -> None:
    mod = _load()
    conn = FakeConn()
    cursor = FakeCursor()
    result = mod.run(cursor, conn, retention_days=90, batch_size=1000,
                     dry_run=False, logger=lambda *_: None)
    assert result == {"sessions": 3, "inbox": 2}
    assert conn.commits == 2, f"expected 2 commits, got {conn.commits}"

    # 会话归档 INSERT 必须先于 DELETE；同批归档带正确数量占位符
    session_ops = [c[0] for c in cursor.calls if "auth_impersonation_session" in c[0]]
    insert_idx = next(i for i, s in enumerate(session_ops)
                      if s.startswith("INSERT INTO auth_impersonation_session_archive"))
    delete_idx = next(i for i, s in enumerate(session_ops)
                      if s.startswith("DELETE FROM auth_impersonation_session"))
    assert insert_idx < delete_idx

    # 会话批归档 SQL 的 IN 占位符数量 = 3（本批 id 数）
    arch = session_ops[insert_idx]
    assert arch.count("%s") == 3, f"session archive IN placeholders wrong: {arch}"


def test_run_dry_run_no_writes() -> None:
    mod = _load()
    conn = FakeConn()
    cursor = FakeCursor()
    result = mod.run(cursor, conn, retention_days=90, batch_size=1000,
                     dry_run=True, logger=lambda *_: None)
    assert result == {"sessions": 3, "inbox": 2}
    assert conn.commits == 0, "dry-run must not commit"
    writes = [c[0] for c in cursor.calls if c[0].startswith(("INSERT", "DELETE"))]
    assert not writes, "dry-run must not execute INSERT/DELETE"


def test_archive_ids_placeholder_expansion() -> None:
    mod = _load()
    cursor = FakeCursor()
    # 直接调用 archive_ids：IN 占位符应按 ids 数量展开
    mod.archive_ids(cursor, mod.archive_sessions_sql(), mod.delete_sessions_sql(), [1, 2, 3])
    insert_sql, insert_params = cursor.calls[-2]
    delete_sql, delete_params = cursor.calls[-1]
    assert insert_sql.count("%s") == 3
    assert insert_params == [1, 2, 3]
    assert delete_sql.count("%s") == 3
    assert delete_params == [1, 2, 3]
