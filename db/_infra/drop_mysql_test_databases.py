#!/usr/bin/env python3
"""Drop leftover OpenTestMySQL schemas without touching production databases.

OpenTestMySQL / OpenTestMySQLCloned create `{service}_test_{8}` (and similar)
databases. Failed DROP or killed `go test` processes leave them on the shared
docker-mysql datadir and can fill a tmpfs workspace.

This tool drops only leftover test schemas. It never DROPs:
- MySQL system schemas
- databases listed in db/registry.yaml (task_auth, task_bill, …)

tpl_* clone templates are rebuildable cache. They are listed on --scan but
dropped only with --drop-templates.

Usage:
  python3 db/_infra/drop_mysql_test_databases.py --scan
  python3 db/_infra/drop_mysql_test_databases.py --fix
  python3 db/_infra/drop_mysql_test_databases.py --fix --drop-templates
"""

from __future__ import annotations

import argparse
import sys
from collections.abc import Iterable, Sequence

import mysql_reset as mr

# Clone templates from dbload.templateDBName — not production.
_TPL_PREFIX = "tpl_"


def is_leftover_test_schema(name: str, registry_dbs: Iterable[str]) -> bool:
    """True for OpenTestMySQL leftovers; false for system and registry DBs."""
    n = str(name).strip()
    if not n or n in mr.SYSTEM_DATABASES:
        return False
    if n in set(registry_dbs):
        return False
    lower = n.lower()
    if lower.startswith("test_"):
        return True
    return "_test_" in lower


def is_clone_template_schema(name: str, registry_dbs: Iterable[str]) -> bool:
    n = str(name).strip()
    if not n or n in mr.SYSTEM_DATABASES or n in set(registry_dbs):
        return False
    return n.lower().startswith(_TPL_PREFIX)


def classify_leftovers(
    existing: Sequence[str],
    registry_dbs: Iterable[str],
    *,
    include_templates: bool,
) -> list[str]:
    names = []
    for raw in existing:
        if is_leftover_test_schema(raw, registry_dbs) or (
            include_templates and is_clone_template_schema(raw, registry_dbs)
        ):
            names.append(mr.validate_db_name(raw))
    return sorted(set(names))


def build_drop_sql(names: Sequence[str]) -> str:
    lines = ["SET SESSION sql_log_bin = 0;"]
    for db in names:
        lines.append(f"DROP DATABASE IF EXISTS `{mr.validate_db_name(db)}`;")
    return "\n".join(lines) + "\n"


def _print(msg: str, *, quiet: bool) -> None:
    if not quiet:
        print(msg)


def main(argv: list[str] | None = None) -> int:
    parser = argparse.ArgumentParser(description=__doc__)
    mode = parser.add_mutually_exclusive_group(required=True)
    mode.add_argument("--scan", action="store_true", help="list leftovers, exit 2 if any")
    mode.add_argument("--fix", action="store_true", help="DROP leftover test schemas")
    parser.add_argument(
        "--drop-templates",
        action="store_true",
        help="also DROP tpl_* clone templates (rebuildable cache)",
    )
    parser.add_argument("--quiet", action="store_true")
    args = parser.parse_args(argv)

    root = mr._monorepo_root()
    try:
        container = mr.find_mysql_container(root)
        existing = mr.list_databases(container)
        registry_dbs = mr.registry_database_names(mr.load_registry(root))
    except (RuntimeError, ValueError, TypeError, OSError) as exc:
        _print(f"[drop-test-db] skip (MySQL unavailable): {exc}", quiet=args.quiet)
        return 0

    test_dbs = classify_leftovers(existing, registry_dbs, include_templates=False)
    tpl_dbs = classify_leftovers(existing, registry_dbs, include_templates=True)
    tpl_only = [n for n in tpl_dbs if n not in test_dbs]
    to_drop = classify_leftovers(
        existing, registry_dbs, include_templates=args.drop_templates
    )

    if args.scan:
        _print(f"[drop-test-db] leftover *_test_* / test_* : {len(test_dbs)}", quiet=args.quiet)
        if test_dbs and not args.quiet:
            for n in test_dbs[:40]:
                print(f"  {n}")
            if len(test_dbs) > 40:
                print(f"  ... and {len(test_dbs) - 40} more")
        _print(f"[drop-test-db] tpl_* templates (cache): {len(tpl_only)}", quiet=args.quiet)
        if test_dbs:
            _print(
                "[drop-test-db] drop with: python3 db/_infra/drop_mysql_test_databases.py --fix",
                quiet=args.quiet,
            )
            return 2
        return 0

    if not to_drop:
        _print("[drop-test-db] nothing to drop", quiet=args.quiet)
        return 0

    _print(f"[drop-test-db] dropping {len(to_drop)} schemas", quiet=args.quiet)
    try:
        mr.mysql_exec(container, build_drop_sql(to_drop))
    except RuntimeError as exc:
        print(f"[drop-test-db] ERROR: {exc}", file=sys.stderr)
        return 1
    _print("[drop-test-db] done", quiet=args.quiet)
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
