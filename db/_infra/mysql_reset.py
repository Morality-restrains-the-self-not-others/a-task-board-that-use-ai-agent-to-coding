#!/usr/bin/env python3
"""Dev reset for the local docker MySQL datadir.

Used by runAll 「清空全部数据库」 via db/_infra/mysql-reset.sh.

Policy (fixes leftover datadir directories + binlog bloat):
- DROP every non-system schema, including OpenTestMySQL leftovers
  (task_*_test_*, test_infra_*, …) that registry.yaml does not list
- CREATE only databases registered in db/registry.yaml
- SET sql_log_bin=0 then RESET MASTER so DROP/CREATE does not grow
  binary logs, and existing binlog.* files are deleted

All mysql commands run through docker exec so the host need not install
the mysql CLI.
"""

from __future__ import annotations

import re
import subprocess
import sys
from collections.abc import Iterable, Sequence
from pathlib import Path

SYSTEM_DATABASES = frozenset(
    {"information_schema", "mysql", "performance_schema", "sys"}
)
_DB_NAME_RE = re.compile(r"^[A-Za-z0-9_-]+$")
COMPOSE_PROJECT = "docker-mysql"
MYSQL_ROOT_USER = "root"
# Local docker-compose MYSQL_ROOT_PASSWORD; never printed.
MYSQL_ROOT_PASSWORD = "root123456"


def _monorepo_root() -> Path:
    return Path(__file__).resolve().parents[2]


def validate_db_name(name: str) -> str:
    if not _DB_NAME_RE.match(name):
        raise ValueError(f"invalid database name: {name!r}")
    return name


def user_databases_to_drop(existing: Iterable[str]) -> list[str]:
    """Every schema except MySQL system schemas, sorted for stable SQL."""
    names = []
    for raw in existing:
        name = str(raw).strip()
        if not name or name in SYSTEM_DATABASES:
            continue
        names.append(validate_db_name(name))
    return sorted(set(names))


def registry_database_names(registry: dict) -> list[str]:
    databases = registry.get("databases") or {}
    names = []
    for key, entry in databases.items():
        if isinstance(entry, dict):
            names.append(str(entry.get("database") or key))
        else:
            names.append(str(key))
    return sorted({validate_db_name(n) for n in names})


def load_registry(root: Path) -> dict:
    import yaml

    path = root / "db" / "registry.yaml"
    data = yaml.safe_load(path.read_text(encoding="utf-8"))
    if not isinstance(data, dict):
        raise TypeError(f"registry.yaml is not a mapping: {path}")
    return data


def build_reset_sql(existing: Sequence[str], registry_dbs: Sequence[str]) -> str:
    """SQL that wipes user schemas, recreates registry DBs, and purges binlogs.

    RESET MASTER runs twice: once before DROP (free disk while tmpfs is tight)
    and once after DDL so any leaked binlog events are also removed.
    """
    drops = user_databases_to_drop(existing)
    recreate = [validate_db_name(n) for n in registry_dbs]
    lines = [
        "CREATE USER IF NOT EXISTS 'taskapp'@'%' IDENTIFIED BY 'taskapp123';",
        "GRANT ALL PRIVILEGES ON *.* TO 'taskapp'@'%' WITH GRANT OPTION;",
        "SET SESSION sql_log_bin = 0;",
        "RESET MASTER;",
    ]
    for db in drops:
        lines.append(f"DROP DATABASE IF EXISTS `{db}`;")
    for db in recreate:
        lines.append(
            f"CREATE DATABASE `{db}` CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;"
        )
        lines.append(f"GRANT ALL PRIVILEGES ON `{db}`.* TO 'taskapp'@'%';")
    lines.append("FLUSH PRIVILEGES;")
    lines.append("RESET MASTER;")
    return "\n".join(lines) + "\n"


KILLABLE_SESSIONS_SQL = (
    "SELECT ID FROM information_schema.PROCESSLIST "
    "WHERE ID != CONNECTION_ID() AND COMMAND != 'Daemon';"
)


def parse_killable_session_ids(stdout: str) -> list[int]:
    ids = []
    for line in stdout.splitlines():
        raw = line.strip()
        if not raw or raw.lower() == "id" or not raw.isdigit():
            continue
        ids.append(int(raw))
    return ids


def kill_other_sessions(container: str, runner=subprocess.run) -> list[int]:
    """Terminate other client sessions so DROP DATABASE is not blocked."""
    raw = mysql_exec(container, KILLABLE_SESSIONS_SQL, runner=runner)
    killed = []
    for sid in parse_killable_session_ids(raw):
        try:
            mysql_exec(container, f"KILL {sid};", runner=runner)
            print(f"[mysql-reset] killed session {sid}")
            killed.append(sid)
        except RuntimeError as exc:
            print(f"[mysql-reset] session {sid} already gone: {exc}")
    return killed


def find_mysql_container(
    root: Path,
    runner=subprocess.run,
) -> str:
    compose = root / "dockerInfra" / "mysql" / "docker-compose.yml"
    ps = runner(
        [
            "docker",
            "compose",
            "-f",
            str(compose),
            "-p",
            COMPOSE_PROJECT,
            "ps",
            "-q",
            "mysql",
        ],
        capture_output=True,
        text=True,
        check=False,
    )
    cid = (ps.stdout or "").strip().split()
    if cid:
        return cid[0]
    ps = runner(
        [
            "docker",
            "ps",
            "--filter",
            "name=docker-mysql",
            "--format",
            "{{.ID}}",
        ],
        capture_output=True,
        text=True,
        check=False,
    )
    cid = (ps.stdout or "").strip().split()
    if cid:
        return cid[0]
    raise RuntimeError(
        "MySQL container not found — is docker-mysql running?"
    )


def mysql_exec(container: str, sql: str, runner=subprocess.run) -> str:
    completed = runner(
        [
            "docker",
            "exec",
            container,
            "mysql",
            "--default-character-set=utf8mb4",
            f"-u{MYSQL_ROOT_USER}",
            f"-p{MYSQL_ROOT_PASSWORD}",
            "-e",
            sql,
        ],
        capture_output=True,
        text=True,
        check=False,
    )
    if completed.returncode != 0:
        err = (completed.stderr or completed.stdout or "").strip()
        raise RuntimeError(f"mysql exec failed: {err}")
    return completed.stdout or ""


def list_databases(container: str, runner=subprocess.run) -> list[str]:
    out = mysql_exec(container, "SHOW DATABASES;", runner=runner)
    names = []
    for line in out.splitlines():
        name = line.strip()
        if not name or name.lower() == "database":
            continue
        names.append(name)
    return names


def main() -> int:
    root = _monorepo_root()
    try:
        container = find_mysql_container(root)
    except RuntimeError as exc:
        print(f"[mysql-reset] ERROR: {exc}", file=sys.stderr)
        return 1
    print(f"[mysql-reset] container={container}")

    try:
        existing = list_databases(container)
        registry = load_registry(root)
        registry_dbs = registry_database_names(registry)
        to_drop = user_databases_to_drop(existing)
        print(
            f"[mysql-reset] {len(to_drop)} user databases to drop "
            f"(registry={len(registry_dbs)}, leftovers="
            f"{len(to_drop) - len(set(to_drop) & set(registry_dbs))}): "
            f"{' '.join(to_drop)}"
        )
        print(
            f"[mysql-reset] recreating {len(registry_dbs)} registry databases: "
            f"{' '.join(registry_dbs)}"
        )
        print("[mysql-reset] killing other client sessions so DROP is not blocked")
        kill_other_sessions(container)
        print("[mysql-reset] purging binary logs with RESET MASTER")
        sql = build_reset_sql(existing, registry_dbs)
        mysql_exec(container, sql)
    except (RuntimeError, ValueError, TypeError, OSError) as exc:
        print(f"[mysql-reset] ERROR: {exc}", file=sys.stderr)
        return 1

    print("[mysql-reset] all databases reset successfully")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
