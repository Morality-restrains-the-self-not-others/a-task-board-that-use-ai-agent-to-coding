#!/usr/bin/env python3
"""archive_impersonation_sessions.py — 归档 90 天外的模拟登录会话与收信箱消息（OPT-20260823-017）。

背景：`auth_impersonation_session` 与 `auth_user_inbox_message` 为时间累积型审计表
（年增量 <10 万，见 dataMigrate/taskAuth/036/037），本期只建热表。热表无限膨胀会
拖慢 `idx_auth_impersonation_actor_open` 等索引与审计查询。

行为：
  - 会话归档条件：`ended_at` 非空且 < 90 天前；或 `ended_at` 为空且 `expires_at` < 90 天前
  - 归档会话时同步归档其关联收信箱消息（`impersonation_session_id`），保持成对
  - 再单独归档 `impersonation_session_id IS NULL` 且 `created_at` < 90 天前的收信箱消息
  - 每批 ≤1000 行，单事务内 INSERT→archive + DELETE→hot，崩溃重放安全（归档表无唯一键）
  - `--dry-run` 只打印不执行；`--self-test` 用 mock 游标跑核心逻辑
  - 打印每轮归档行数，供 cron 日志 / 告警

使用方式：
  python3 db/scripts/archive_impersonation_sessions.py
  python3 db/scripts/archive_impersonation_sessions.py --retention-days 90 --batch-size 1000
  python3 db/scripts/archive_impersonation_sessions.py --dry-run

环境变量（与 db/scripts/apply_datamigrate.sh 对齐）：
  MYSQL_HOST / MYSQL_PORT / MYSQL_USER / MYSQL_PASS  连接配置（默认 10.2.150.68 taskapp/taskapp123）
  MYSQL_DATABASE=task_auth                            目标库（默认 task_auth）

Cron 建议（每周）：0 4 * * 0 cd /tmp/ram-work && python3 db/scripts/archive_impersonation_sessions.py >> logs/archive-impersonation-cron.log 2>&1
"""
from __future__ import annotations

import argparse
import os
import sys

try:
    import pymysql
except ImportError:
    print("ERROR: pymysql required. Run: pip install pymysql", file=sys.stderr)
    sys.exit(1)

DEFAULT_DB = "task_auth"
DEFAULT_HOST = os.environ.get("MYSQL_HOST", "10.2.150.68")
DEFAULT_PORT = int(os.environ.get("MYSQL_PORT", "3306"))
DEFAULT_USER = os.environ.get("MYSQL_USER", "taskapp")
DEFAULT_PASS = os.environ.get("MYSQL_PASS", "taskapp123")

# 热表列（归档表 = 热表列 + archived_at，见 dataMigrate/taskAuth/041）。
SESSION_COLUMNS = [
    "id", "actor_user_id", "target_user_id", "token_key", "idempotency_key",
    "reason", "started_at", "expires_at", "ended_at",
]
INBOX_COLUMNS = [
    "id", "recipient_user_id", "kind", "title", "body", "reason",
    "actor_user_id", "impersonation_session_id", "created_at", "read_at",
]


def placeholders(n: int) -> str:
    return ",".join(["%s"] * n)


def select_session_ids_sql(retention_days: int, batch_size: int) -> tuple[str, tuple]:
    """选出待归档会话 id：ended_at 超期 或 未结束但 expires_at 超期。"""
    sql = (
        "SELECT id FROM auth_impersonation_session "
        "WHERE (ended_at IS NOT NULL AND ended_at < (UTC_TIMESTAMP() - INTERVAL %s DAY)) "
        "   OR (ended_at IS NULL AND expires_at < (UTC_TIMESTAMP() - INTERVAL %s DAY)) "
        "ORDER BY id LIMIT %s"
    )
    return sql, (retention_days, retention_days, batch_size)


def select_orphan_inbox_ids_sql(retention_days: int, batch_size: int) -> tuple[str, tuple]:
    """选出不与任何会话关联的超期收信箱消息 id。"""
    sql = (
        "SELECT id FROM auth_user_inbox_message "
        "WHERE created_at < (UTC_TIMESTAMP() - INTERVAL %s DAY) "
        "  AND impersonation_session_id IS NULL "
        "ORDER BY id LIMIT %s"
    )
    return sql, (retention_days, batch_size)


def archive_inbox_for_sessions_sql() -> str:
    cols = ", ".join(INBOX_COLUMNS)
    return (
        f"INSERT INTO auth_user_inbox_message_archive ({cols}, archived_at) "
        f"SELECT {cols}, UTC_TIMESTAMP() FROM auth_user_inbox_message "
        f"WHERE impersonation_session_id IN ({placeholders(len(INBOX_COLUMNS) and 1)})"
    )


def delete_inbox_for_sessions_sql() -> str:
    return (
        "DELETE FROM auth_user_inbox_message "
        "WHERE impersonation_session_id IN (%s)"
    )


def archive_sessions_sql() -> str:
    cols = ", ".join(SESSION_COLUMNS)
    return (
        f"INSERT INTO auth_impersonation_session_archive ({cols}, archived_at) "
        f"SELECT {cols}, UTC_TIMESTAMP() FROM auth_impersonation_session "
        f"WHERE id IN ({placeholders(len(SESSION_COLUMNS) and 1)})"
    )


def delete_sessions_sql() -> str:
    return "DELETE FROM auth_impersonation_session WHERE id IN (%s)"


def archive_orphan_inbox_sql() -> str:
    cols = ", ".join(INBOX_COLUMNS)
    return (
        f"INSERT INTO auth_user_inbox_message_archive ({cols}, archived_at) "
        f"SELECT {cols}, UTC_TIMESTAMP() FROM auth_user_inbox_message "
        f"WHERE id IN ({placeholders(len(INBOX_COLUMNS) and 1)})"
    )


def delete_orphan_inbox_sql() -> str:
    return "DELETE FROM auth_user_inbox_message WHERE id IN (%s)"


def archive_ids(cursor, archive_sql, delete_sql, ids):
    """单批归档：INSERT→archive + DELETE→hot。返回归档行数。"""
    ph = placeholders(len(ids))
    cursor.execute(archive_sql.replace("(%s)", f"({ph})", 1), ids)
    cursor.execute(delete_sql.replace("(%s)", f"({ph})", 1), ids)
    return len(ids)


def run(cursor, conn, retention_days: int = 90, batch_size: int = 1000,
        dry_run: bool = False, logger=print) -> dict:
    """归档超期会话与其收信箱消息。返回 {sessions, inbox} 计数。"""
    total_sessions = 0
    total_inbox = 0

    # 1) 会话（连带其收信箱消息）
    while True:
        sql, params = select_session_ids_sql(retention_days, batch_size)
        cursor.execute(sql, params)
        session_ids = [row[0] for row in cursor.fetchall()]
        if not session_ids:
            break
        if dry_run:
            logger(f"[dry-run] 会话批 {len(session_ids)} 行：将归档 auth_impersonation_session + 关联 inbox")
        else:
            archive_ids(cursor, archive_sessions_sql(), delete_sessions_sql(), session_ids)
            archive_ids(cursor, archive_inbox_for_sessions_sql(), delete_inbox_for_sessions_sql(), session_ids)
            conn.commit()
        total_sessions += len(session_ids)

    # 2) 无会话关联的收信箱消息（created_at 超期）
    while True:
        sql, params = select_orphan_inbox_ids_sql(retention_days, batch_size)
        cursor.execute(sql, params)
        inbox_ids = [row[0] for row in cursor.fetchall()]
        if not inbox_ids:
            break
        if dry_run:
            logger(f"[dry-run] 收信箱批 {len(inbox_ids)} 行：将归档 auth_user_inbox_message（无会话关联）")
        else:
            archive_ids(cursor, archive_orphan_inbox_sql(), delete_orphan_inbox_sql(), inbox_ids)
            conn.commit()
        total_inbox += len(inbox_ids)

    return {"sessions": total_sessions, "inbox": total_inbox}


def main(argv: list[str] | None = None) -> int:
    ap = argparse.ArgumentParser(description=__doc__)
    ap.add_argument("--retention-days", type=int, default=int(os.environ.get("RETENTION_DAYS", "90")),
                    help="归档阈值：ended_at/expires_at/created_at 距今超过该天数即归档（默认 90）")
    ap.add_argument("--batch-size", type=int, default=int(os.environ.get("BATCH_SIZE", "1000")),
                    help="每批最大行数（默认 1000，OPT 约束 ≤1000）")
    ap.add_argument("--database", default=os.environ.get("MYSQL_DATABASE", DEFAULT_DB),
                    help="目标数据库（默认 task_auth）")
    ap.add_argument("--dry-run", action="store_true", help="只打印不执行")
    ap.add_argument("--self-test", action="store_true", help="用 mock 游标跑核心逻辑（不连库）")
    args = ap.parse_args(argv)

    if args.self_test:
        return _self_test(args.retention_days, args.batch_size)

    conn = pymysql.connect(
        host=DEFAULT_HOST, port=DEFAULT_PORT, user=DEFAULT_USER,
        password=DEFAULT_PASS, database=args.database, charset="utf8mb4",
    )
    try:
        with conn.cursor() as cursor:
            result = run(cursor, conn, retention_days=args.retention_days,
                         batch_size=args.batch_size, dry_run=args.dry_run)
        print(f"archive done: sessions={result['sessions']} inbox={result['inbox']} "
              f"(retention={args.retention_days}d, dry_run={args.dry_run})")
        return 0
    except Exception as e:
        print(f"ERROR: 归档 {args.database} 失败: {e}", file=sys.stderr)
        try:
            conn.rollback()
        except Exception:
            pass
        return 2


class _FakeCursor:
    """self-test 用游标：session 查询返回一批 id，随后归档/删除仅记录调用。"""

    def __init__(self):
        self.calls = []
        self._phase = 0

    def execute(self, sql, params=None):
        self.calls.append((sql, params))
        self._sql = sql
        self._params = params or ()

    def fetchall(self):
        if "FROM auth_impersonation_session" in self._sql and "archive" not in self._sql:
            # 第一轮返回 3 个会话 id，第二轮返回空 → 结束会话循环
            if self._phase == 0:
                self._phase = 1
                return [(11,), (12,), (13,)]
            return []
        if "FROM auth_user_inbox_message" in self._sql and "archive" not in self._sql:
            # 孤儿收信箱第一轮返回 2 个 id，第二轮空
            if self._phase == 1:
                self._phase = 2
                return [(21,), (22,)]
            return []
        return []


class _FakeConn:
    def __init__(self):
        self.commits = 0

    def cursor(self):
        return _FakeCursor()

    def commit(self):
        self.commits += 1

    def rollback(self):
        pass


def _self_test(retention_days: int, batch_size: int) -> int:
    logs: list[str] = []
    conn = _FakeConn()
    cursor = _FakeCursor()

    # 1) 正常归档
    result = run(cursor, conn, retention_days=retention_days, batch_size=batch_size,
                 dry_run=False, logger=logs.append)
    assert result == {"sessions": 3, "inbox": 2}, f"unexpected result: {result}"
    assert conn.commits == 2, f"expected 2 commits (session batch + inbox batch), got {conn.commits}"

    # 校验会话归档先于删除（INSERT 在 DELETE 之前）
    session_ops = [c[0] for c in cursor.calls if "auth_impersonation_session" in c[0]]
    insert_idx = next(i for i, s in enumerate(session_ops) if s.startswith("INSERT INTO auth_impersonation_session_archive"))
    delete_idx = next(i for i, s in enumerate(session_ops) if s.startswith("DELETE FROM auth_impersonation_session"))
    assert insert_idx < delete_idx, "session archive INSERT must precede DELETE"

    # 2) dry-run 不执行任何写
    cursor2 = _FakeCursor()
    conn2 = _FakeConn()
    logs2: list[str] = []
    result2 = run(cursor2, conn2, retention_days=retention_days, batch_size=batch_size,
                  dry_run=True, logger=logs2.append)
    assert result2 == {"sessions": 3, "inbox": 2}, f"dry-run result mismatch: {result2}"
    assert conn2.commits == 0, "dry-run must not commit"
    writes = [c[0] for c in cursor2.calls if c[0].startswith(("INSERT", "DELETE"))]
    assert not writes, "dry-run must not execute INSERT/DELETE"

    # 3) SQL 生成
    ssql, sp = select_session_ids_sql(retention_days, batch_size)
    assert ssql.count("%s") == 3 and sp == (retention_days, retention_days, batch_size)
    isql, ip = select_orphan_inbox_ids_sql(retention_days, batch_size)
    assert ip == (retention_days, batch_size)

    print("self-test ok: run batches/dry-run/SQL generation all pass")
    return 0


if __name__ == "__main__":
    sys.exit(main())
