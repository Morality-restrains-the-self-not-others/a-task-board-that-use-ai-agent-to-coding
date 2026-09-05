#!/usr/bin/env python3
"""Backfill system-auto git identities for existing tenant members.

OPT-20260812-028: the MEMBER_JOINED → ensure-default consumer only covers
newly joined members; historical members may lack a `system-auto` git identity
(task_task.task_git_identities), which breaks task cloning when no default
company identity exists.

This script scans tenant_company_member (task_tenant DB) and calls taskTaskService's
idempotent POST /api/internal/git-identities/ensure-default/ for members that do
not yet have a system-auto identity for their company.

Usage:
    python3 db/_infra/backfill_system_auto_git_identities.py               # dry-run (default)
    python3 db/_infra/backfill_system_auto_git_identities.py --confirm RUN  # execute
"""

from __future__ import annotations

import argparse
import json
import os
import urllib.request

try:
    import pymysql
except ImportError:  # pragma: no cover
    raise SystemExit("pymysql is required: pip install pymysql")

MYSQL_HOST = os.environ.get("MYSQL_HOST", "10.2.150.68")
MYSQL_PORT = int(os.environ.get("MYSQL_PORT", "3306"))
MYSQL_USER = os.environ.get("MYSQL_USER", "root")
MYSQL_PASSWORD = os.environ.get("MYSQL_PASSWORD", "root123456")
TASK_TASK_BASE_URL = os.environ.get("TASK_TASK_SERVICE_BASE_URL", "http://127.0.0.1:8017")
INTERNAL_SECRET = os.environ.get("SHARED_INTERNAL_SECRET", "") or os.environ.get("TASK_TASK_INTERNAL_SECRET", "")


def _connect(database: str):
    return pymysql.connect(
        host=MYSQL_HOST,
        port=MYSQL_PORT,
        user=MYSQL_USER,
        password=MYSQL_PASSWORD,
        database=database,
        charset="utf8mb4",
        connect_timeout=10,
    )


def _ensure_default(user_id, company_id, member_id, member_name, confirm) -> tuple[str, str]:
    """Call taskTaskService ensure-default; returns (kind, detail)."""
    body = json.dumps({
        "user_id": user_id,
        "company_id": company_id,
        "member_id": member_id,
        "member_name": member_name,
    }).encode("utf-8")
    req = urllib.request.Request(
        TASK_TASK_BASE_URL.rstrip("/") + "/api/internal/git-identities/ensure-default/",
        data=body,
        method="POST",
        headers={"Content-Type": "application/json", "X-Auth-User-Id": "internal"},
    )
    if INTERNAL_SECRET:
        req.add_header("X-Internal-Secret", INTERNAL_SECRET)
    try:
        with urllib.request.urlopen(req, timeout=20) as resp:
            raw = resp.read().decode("utf-8", "replace")[:300]
            if not confirm:
                return "would-ensure", f"status={resp.status} {raw}"
            return ("created" if resp.status == 201 else "ok"), f"status={resp.status} {raw}"
    except urllib.error.HTTPError as e:
        return "failed", f"status={e.code} {e.read().decode('utf-8', 'replace')[:300]}"
    except Exception as e:  # noqa: BLE001
        return "failed", f"{e}"


def main() -> int:
    ap = argparse.ArgumentParser(description="Backfill system-auto git identities for existing members")
    ap.add_argument("--confirm", default="", help="must be RUN to actually ensure identities")
    args = ap.parse_args()
    confirm = args.confirm == "RUN"

    members = []
    with _connect("task_tenant") as conn:
        cur = conn.cursor()
        cur.execute(
            "SELECT id, user_id, company_id, COALESCE(member_name,'') "
            "FROM tenant_company_member"
        )
        members = list(cur.fetchall())

    have = set()
    with _connect("task_task") as conn:
        cur = conn.cursor()
        cur.execute("SELECT user_id, company_id FROM task_git_identities WHERE label='system-auto'")
        have = {(str(r[0]), str(r[1])) for r in cur.fetchall()}

    missing = [m for m in members if (str(m[1]), str(m[2])) not in have]
    print(f"members={len(members)} with_system_auto={len(have)} missing={len(missing)}")

    counts = {"created": 0, "ok": 0, "would-ensure": 0, "failed": 0}
    for member_id, user_id, company_id, member_name in missing:
        kind, detail = _ensure_default(str(user_id), str(company_id), str(member_id), member_name, confirm)
        counts[kind] = counts.get(kind, 0) + 1
        print(f"[{kind}] member={member_id} user={user_id} company={company_id} -> {detail}")

    print(f"done confirm={confirm} missing={len(missing)} counts={counts}")
    return 0 if counts["failed"] == 0 else 1


if __name__ == "__main__":
    raise SystemExit(main())
