#!/usr/bin/env python3
"""Dry-run expand/dual-write/contract plan for overflow negative IDs (OPT-20260821-013).

Does not apply DDL or rewrite primary keys. Live `UPDATE ... SET id=` is forbidden.
"""
from __future__ import annotations

import argparse
import re
import sys

OVERFLOW_WORKSPACES = (
    "ws_-2309487803472456748",  # tenant 877397588196749312
    "ws_-2740859684112864748",  # tenant 875588283562749952
)
OVERFLOW_PROJECTS = (
    "proj_-2304947540687519745",  # tenant 877397588196749312
)

OVERFLOW_RE = re.compile(r"^(ws|proj)_-\d+$")
PK_REWRITE_RE = re.compile(
    r"UPDATE\s+project_(?:workspace_entries|entries)\s+SET\s+id\s*=",
    re.IGNORECASE,
)

EXPAND_SQL = """
-- EXPAND (plan only — do not apply from this script)
-- Future dataMigrate/taskProjectService/NNN_id_alias.sql would create:
CREATE TABLE IF NOT EXISTS project_id_alias (
  kind VARCHAR(32) NOT NULL,
  canonical_id VARCHAR(64) NOT NULL,
  stored_id VARCHAR(64) NOT NULL,
  created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (kind, canonical_id),
  UNIQUE KEY uk_project_id_alias_stored (kind, stored_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
""".strip()


def inventory_lines() -> list[str]:
    lines = [
        f"INVENTORY workspaces={len(OVERFLOW_WORKSPACES)} projects={len(OVERFLOW_PROJECTS)}",
    ]
    for wid in OVERFLOW_WORKSPACES:
        if not OVERFLOW_RE.match(wid):
            raise SystemExit(f"invalid workspace inventory id: {wid}")
        lines.append(f"  workspace stored_pk={wid}")
    for pid in OVERFLOW_PROJECTS:
        if not OVERFLOW_RE.match(pid):
            raise SystemExit(f"invalid project inventory id: {pid}")
        lines.append(f"  project stored_pk={pid}")
    return lines


def plan_text() -> str:
    dual_write = """
DUAL-WRITE
  1. Keep overflow strings as stored PKs (opaque VARCHAR). Never ParseInt the suffix.
  2. Register canonical snowflake → stored PK in-process (taskProjectService idAliases)
     and later in project_id_alias. GET/PUT/DELETE by either id hits the same row.
  3. JSON `id` remains the stored overflow PK so old URLs keep working.
  4. New rows continue to use genID → positive snowflake (no ws_-/proj_-).
""".strip()
    contract = """
CONTRACT (later, not this script)
  1. Copy overflow row to a new snowflake PK; point alias canonical=stored_new.
  2. Dual-write FKs (project_workspaces, accesses) to the new PK.
  3. Switch reads to the new PK only after clients stop sending overflow URLs.
  4. Drop overflow row last. Forbidden: lock-table UPDATE of the primary key.
""".strip()
    text = "\n".join(
        [
            "OPT-20260821-013 negative ID migration plan (--check, no apply)",
            *inventory_lines(),
            "",
            "EXPAND",
            EXPAND_SQL,
            "",
            dual_write,
            "",
            contract,
        ]
    )
    if PK_REWRITE_RE.search(text):
        raise SystemExit("plan unexpectedly contains PK rewrite UPDATE")
    return text


def main(argv: list[str] | None = None) -> int:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument(
        "--check",
        action="store_true",
        help="Print expand/dual-write/contract plan and exit 0",
    )
    parser.add_argument(
        "--apply",
        action="store_true",
        help="Rejected: this script never rewrites primary keys",
    )
    args = parser.parse_args(argv)
    if args.apply:
        print("refuses live PK UPDATE; use --check only", file=sys.stderr)
        return 2
    if not args.check:
        parser.print_help()
        return 2
    print(plan_text())
    return 0


if __name__ == "__main__":
    sys.exit(main())
