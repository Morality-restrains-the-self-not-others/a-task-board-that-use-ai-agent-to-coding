#!/usr/bin/env python3
"""Strip SQLite REFERENCES to accounts_company from saas child tables.

accounts_company is owned by task-tenant; saas child tables keep company_id
columns but must not declare FK constraints to the dropped parent table.
"""

from __future__ import annotations

import argparse
import re
import sqlite3
import sys
from pathlib import Path

PARENT = "accounts_company"
_REF_RE = re.compile(
    r'\s*REFERENCES\s+"?' + re.escape(PARENT) + r'"?\s*\([^)]+\)'
    r"(?:\s+DEFERRABLE\s+INITIALLY\s+DEFERRED)?",
    re.IGNORECASE,
)


def _tables_with_parent_fk(conn: sqlite3.Connection) -> list[tuple[str, str]]:
    rows = conn.execute(
        "SELECT name, sql FROM sqlite_master WHERE type='table' AND sql IS NOT NULL"
    ).fetchall()
    out: list[tuple[str, str]] = []
    for name, sql in rows:
        if name.startswith("sqlite_"):
            continue
        if _REF_RE.search(sql):
            out.append((name, sql))
    return out


def _indexes_for_table(conn: sqlite3.Connection, table: str) -> list[str]:
    rows = conn.execute(
        "SELECT sql FROM sqlite_master "
        "WHERE type='index' AND tbl_name=? AND sql IS NOT NULL",
        (table,),
    ).fetchall()
    return [r[0] for r in rows if r[0]]


def strip_accounts_company_fk_on_connection(conn: sqlite3.Connection) -> list[str]:
    """Rebuild tables that REFERENCE accounts_company without that FK. Returns touched names."""
    conn.execute("PRAGMA foreign_keys=OFF")
    touched: list[str] = []
    for name, sql in _tables_with_parent_fk(conn):
        new_sql = _REF_RE.sub("", sql)
        if new_sql == sql:
            continue
        create_new = re.sub(
            r'(?i)^(CREATE\s+TABLE\s+)("?)' + re.escape(name) + r"\2",
            rf"\1\2{name}__nofk\2",
            new_sql,
            count=1,
        )
        if create_new == new_sql:
            raise RuntimeError(f"failed to rewrite CREATE TABLE for {name}")

        indexes = _indexes_for_table(conn, name)
        cols = [r[1] for r in conn.execute(f'PRAGMA table_info("{name}")').fetchall()]
        col_list = ", ".join(f'"{c}"' for c in cols)

        # Django migrations already open a transaction; avoid nested BEGIN.
        own_tx = not conn.in_transaction
        if own_tx:
            conn.execute("BEGIN")
        try:
            conn.execute(create_new)
            conn.execute(
                f'INSERT INTO "{name}__nofk" ({col_list}) '
                f'SELECT {col_list} FROM "{name}"'
            )
            conn.execute(f'DROP TABLE "{name}"')
            conn.execute(f'ALTER TABLE "{name}__nofk" RENAME TO "{name}"')
            for idx_sql in indexes:
                conn.execute(idx_sql)
            if own_tx:
                conn.execute("COMMIT")
        except Exception:
            if own_tx:
                conn.execute("ROLLBACK")
            raise
        touched.append(name)

    conn.execute("PRAGMA foreign_keys=ON")
    remaining = _tables_with_parent_fk(conn)
    if remaining:
        names = ", ".join(n for n, _ in remaining)
        raise RuntimeError(f"still REFERENCE {PARENT}: {names}")
    return touched


def strip_accounts_company_fk(db_path: str | Path) -> list[str]:
    """Open a file-backed sqlite DB and strip accounts_company FK REFERENCES."""
    path = Path(db_path)
    if not path.is_file():
        raise FileNotFoundError(path)

    conn = sqlite3.connect(str(path))
    try:
        return strip_accounts_company_fk_on_connection(conn)
    finally:
        conn.close()


def main(argv: list[str] | None = None) -> int:
    repo = Path(__file__).resolve().parents[2]
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument(
        "db",
        nargs="?",
        default=str(repo / "db" / "saas" / "saas.sqlite3"),
        help="path to saas.sqlite3",
    )
    args = parser.parse_args(argv)
    touched = strip_accounts_company_fk(args.db)
    if touched:
        print("stripped FK from:", ", ".join(touched))
    else:
        print("ok: no REFERENCES accounts_company remaining")
    return 0


if __name__ == "__main__":
    sys.exit(main())
