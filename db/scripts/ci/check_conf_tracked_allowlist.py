#!/usr/bin/env python3
"""CI: conf.git tracked files must match the config-repo allowlist.

Design: docs/superpowers/specs/2026-09-01-conf-non-config-purge-design.md
"""

from __future__ import annotations

import os
import subprocess
import sys
from pathlib import Path

ALLOWED_EXACT = frozenset(
    {
        ".gitignore",
        "LICENSE",
        "README.md",
        "ai.md",
        ".githooks/pre-commit",
    }
)


def git_clean_env() -> dict[str, str]:
    """Ignore caller GIT_INDEX_FILE/GIT_DIR (e.g. another repo's pre-commit)."""
    env = os.environ.copy()
    for key in (
        "GIT_DIR",
        "GIT_INDEX_FILE",
        "GIT_WORK_TREE",
        "GIT_OBJECT_DIRECTORY",
        "GIT_PREFIX",
        "GIT_COMMON_DIR",
    ):
        env.pop(key, None)
    return env


def monorepo_root() -> Path:
    here = Path(__file__).resolve()
    for parent in here.parents:
        if (parent / ".gitmodules").is_file() and (parent / "db" / "registry.yaml").is_file():
            return parent
    raise FileNotFoundError(".gitmodules + db/registry.yaml not found")


def is_allowed(rel: str) -> bool:
    rel = rel.replace("\\", "/")
    if rel.startswith("./"):
        rel = rel[2:]
    name = Path(rel).name
    if name.startswith("docker-compose") and name.endswith((".yml", ".yaml")):
        return False
    if rel.startswith(".githooks/") and rel != ".githooks/pre-commit":
        return False
    if rel.startswith(".claude/"):
        return False
    if rel in ALLOWED_EXACT:
        return True
    if name.endswith(".yaml") or name.endswith(".yml"):
        return True
    if name.endswith(".ai.md") or name == "ai.md":
        return True
    if name.endswith(".example"):
        return True
    if name == "sync.sh":
        return True
    return False


def collect_violations_from(rels: list[str]) -> list[str]:
    return [rel for rel in rels if not is_allowed(rel)]


def list_conf_tracked(root: Path) -> list[str]:
    conf = root / "conf"
    out = subprocess.check_output(
        ["git", "-C", str(conf), "ls-files"],
        text=True,
        env=git_clean_env(),
    )
    return [line.strip() for line in out.splitlines() if line.strip()]


def main() -> int:
    root = monorepo_root()
    hits = collect_violations_from(list_conf_tracked(root))
    if hits:
        print("FAIL check_conf_tracked_allowlist:", file=sys.stderr)
        for rel in hits:
            print(f"  - {rel}", file=sys.stderr)
        return 1
    print("OK check_conf_tracked_allowlist")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
