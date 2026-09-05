#!/usr/bin/env python3
"""
Root-level compatibility wrapper.

Some skills invoke:
  python runAll/scripts/ci/check_ddd_bdd_compliance.py
from workspace root. The real checker currently lives under:
  task2app/scripts/ci/check_ddd_bdd_compliance.py
This wrapper forwards all CLI args to that script, and also runs
Go DDD compliance for valueStream domain.
"""

from __future__ import annotations

import subprocess
import sys
from pathlib import Path

GO_ONLY_FLAGS = {
    "--module-root",
    "--forbid-external",
    "--strict-module-root",
}


def split_python_and_go_args(argv: list[str]) -> tuple[list[str], list[str]]:
    """
    Split CLI args for:
    - python checker: excludes Go-only flags
    - go checker: receives all args
    """
    python_args: list[str] = []
    i = 0
    while i < len(argv):
        arg = argv[i]
        if arg.startswith("--module-root="):
            i += 1
            continue
        if arg in GO_ONLY_FLAGS:
            if arg == "--module-root" and i + 1 < len(argv):
                i += 2
                continue
            i += 1
            continue
        python_args.append(arg)
        i += 1
    return python_args, list(argv)


def main() -> int:
    root = Path(__file__).resolve().parents[3]
    python_target = root / "task2app" / "scripts" / "ci" / "check_ddd_bdd_compliance.py"
    go_target = root / "valueStream" / "scripts" / "ci" / "check_go_ddd_compliance.py"
    python_args, go_args = split_python_and_go_args(sys.argv[1:])

    first_failure = 0

    if python_target.is_file():
        cmd = [sys.executable, str(python_target), *python_args]
        completed = subprocess.run(cmd, cwd=python_target.parents[2], check=False)
        if completed.returncode != 0 and first_failure == 0:
            first_failure = completed.returncode
    else:
        sys.stderr.write(
            f"python checker not found: {python_target}\n"
            "expected task2app/scripts/ci/check_ddd_bdd_compliance.py\n"
        )
        if first_failure == 0:
            first_failure = 2

    if go_target.is_file():
        cmd = [sys.executable, str(go_target), *go_args]
        completed = subprocess.run(cmd, cwd=root, check=False)
        if completed.returncode != 0 and first_failure == 0:
            first_failure = completed.returncode
    else:
        sys.stderr.write(
            f"go checker not found: {go_target}\n"
            "expected valueStream/scripts/ci/check_go_ddd_compliance.py\n"
        )
        if first_failure == 0:
            first_failure = 2

    conf_sync = root / "conf" / "scripts" / "ci" / "check_conf_sync.sh"
    if conf_sync.is_file():
        completed = subprocess.run(["bash", str(conf_sync)], cwd=root, check=False)
        if completed.returncode != 0 and first_failure == 0:
            first_failure = completed.returncode

    db_ownership = root / "db" / "scripts" / "ci" / "check_single_service_db_ownership.py"
    if db_ownership.is_file():
        completed = subprocess.run(
            [sys.executable, str(db_ownership)], cwd=root, check=False
        )
        if completed.returncode != 0 and first_failure == 0:
            first_failure = completed.returncode
    else:
        sys.stderr.write(
            f"db ownership checker not found: {db_ownership}\n"
            "expected db/scripts/ci/check_single_service_db_ownership.py\n"
        )
        if first_failure == 0:
            first_failure = 2

    api_routes = root / "db" / "scripts" / "ci" / "check_django_new_api_routes.py"
    if api_routes.is_file():
        completed = subprocess.run(
            [sys.executable, str(api_routes)], cwd=root, check=False
        )
        if completed.returncode != 0 and first_failure == 0:
            first_failure = completed.returncode
    else:
        sys.stderr.write(
            f"django API route checker not found: {api_routes}\n"
            "expected db/scripts/ci/check_django_new_api_routes.py\n"
        )
        if first_failure == 0:
            first_failure = 2

    value_stream_src = root / "valueStream" / "src"
    if value_stream_src.is_dir():
        cmd = [
            "go",
            "test",
            "-count=1",
            "./...",
            "-run",
            "TestLoadProductionValueStream|TestDesignDocFieldNames",
        ]
        completed = subprocess.run(cmd, cwd=value_stream_src, check=False)
        if completed.returncode != 0 and first_failure == 0:
            first_failure = completed.returncode

    return first_failure


if __name__ == "__main__":
    raise SystemExit(main())

