"""Load conf bootstrapAdmin.email and render dataMigrate SQL placeholders.

SSOT: conf/auth/task-auth/config.yaml bootstrapAdmin.email（叠 conf-local）。
022_seed_bootstrap_admin.sql uses __BOOTSTRAP_ADMIN_EMAIL__; apply_datamigrate.sh
and taskAuth runDataMigrateFromDir substitute before Exec.
"""
from __future__ import annotations

import sys
from pathlib import Path

PLACEHOLDER = "__BOOTSTRAP_ADMIN_EMAIL__"
_UNSAFE = set("'\\\"\n\r\x00;")


def find_repo_root(start: Path | None = None) -> Path:
    cur = (start or Path(__file__)).resolve()
    for p in [cur.parent, *cur.parents]:
        if (p / "conf" / "auth" / "task-auth" / "config.yaml").is_file():
            return p
    raise FileNotFoundError("conf/auth/task-auth/config.yaml not found from " + str(cur))


def load_bootstrap_admin_email(repo_root: Path | None = None) -> str:
    root = repo_root or find_repo_root()
    scripts = root / "runAll" / "scripts"
    if not scripts.is_dir():
        scripts = find_repo_root() / "runAll" / "scripts"
    if str(scripts) not in sys.path:
        sys.path.insert(0, str(scripts))
    from conf_local import overlay_conf_file

    cfg = overlay_conf_file(root / "conf" / "auth" / "task-auth" / "config.yaml")
    email = str(((cfg or {}).get("bootstrapAdmin") or {}).get("email") or "").strip()
    return assert_sql_safe_email(email)


def assert_sql_safe_email(email: str) -> str:
    email = (email or "").strip()
    if not email or "@" not in email:
        raise ValueError("bootstrapAdmin.email missing or invalid in conf/auth/task-auth")
    if any(c in email for c in _UNSAFE):
        raise ValueError("bootstrapAdmin.email contains SQL-unsafe characters")
    return email


def render_sql(text: str, email: str) -> str:
    if PLACEHOLDER not in text:
        return text
    return text.replace(PLACEHOLDER, assert_sql_safe_email(email))


def main(argv: list[str]) -> int:
    if len(argv) < 3 or argv[1] != "render":
        print("usage: bootstrap_admin_email.py render <sql-file>", file=sys.stderr)
        return 2
    path = Path(argv[2])
    email = load_bootstrap_admin_email()
    sys.stdout.write(render_sql(path.read_text(encoding="utf-8"), email))
    return 0


if __name__ == "__main__":
    raise SystemExit(main(sys.argv))
