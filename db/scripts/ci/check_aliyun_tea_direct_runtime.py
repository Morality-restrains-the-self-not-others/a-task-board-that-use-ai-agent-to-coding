#!/usr/bin/env python3
"""Forbid hand-written Aliyun Tea RuntimeOptions(); require direct_runtime_options().

SSOT: .ai/01_project_constraints/23_app_startup_no_env_proxy.md
Helper: task2app/Saas_project/cloud/providers/aliyun/direct_network.py

Exit codes:
  0 — no violations
  1 — hand-written RuntimeOptions(...) found outside allowlist
  2 — configuration / IO error
"""

from __future__ import annotations

import argparse
import re
import sys
from pathlib import Path

SKIP_DIR_NAMES = {
    ".git",
    "node_modules",
    "vendor",
    "__pycache__",
    ".venv",
    "venv",
    "dist",
    "build",
    ".tox",
}

# Only these files may construct RuntimeOptions(...).
ALLOWLIST_SUFFIXES = (
    "/cloud/providers/aliyun/direct_network.py",
    "/db/scripts/ci/check_aliyun_tea_direct_runtime.py",
    "/db/scripts/ci/test_check_aliyun_tea_direct_runtime.py",
)

# Matches RuntimeOptions( and util_models.RuntimeOptions(
RUNTIME_OPTIONS_CALL = re.compile(
    r"(?:(?P<mod>\w+)\.)?RuntimeOptions\s*\(",
)


def monorepo_root() -> Path:
    here = Path(__file__).resolve()
    for parent in here.parents:
        if (parent / "db" / "registry.yaml").is_file():
            return parent
    raise FileNotFoundError("db/registry.yaml not found above checker")


def is_allowlisted(path: Path, root: Path) -> bool:
    rel = "/" + path.resolve().relative_to(root).as_posix()
    return any(rel.endswith(suf) for suf in ALLOWLIST_SUFFIXES)


def iter_python_files(root: Path) -> list[Path]:
    # Primary surface: Django Saas_project + related scripts under task2app.
    search_roots = [
        root / "task2app" / "Saas_project",
    ]
    out: list[Path] = []
    for base in search_roots:
        if not base.is_dir():
            continue
        for path in base.rglob("*.py"):
            if any(part in SKIP_DIR_NAMES for part in path.parts):
                continue
            out.append(path)
    return sorted(out)


def find_violations(root: Path) -> list[str]:
    violations: list[str] = []
    for path in iter_python_files(root):
        if is_allowlisted(path, root):
            continue
        try:
            text = path.read_text(encoding="utf-8")
        except OSError as exc:
            raise RuntimeError(f"read failed: {path}: {exc}") from exc
        for lineno, line in enumerate(text.splitlines(), start=1):
            stripped = line.lstrip()
            if stripped.startswith("#"):
                continue
            match = RUNTIME_OPTIONS_CALL.search(line)
            if not match:
                continue
            rel = path.relative_to(root).as_posix()
            violations.append(
                f"{rel}:{lineno}: hand-written RuntimeOptions() — "
                f"use cloud.providers.aliyun.direct_network.direct_runtime_options() instead\n"
                f"  {line.strip()}"
            )
    return violations


def main(argv: list[str] | None = None) -> int:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument(
        "--root",
        type=Path,
        default=None,
        help="Monorepo root (default: auto-detect)",
    )
    args = parser.parse_args(argv)
    try:
        root = (args.root or monorepo_root()).resolve()
    except FileNotFoundError as exc:
        print(f"ERROR: {exc}", file=sys.stderr)
        return 2

    try:
        violations = find_violations(root)
    except RuntimeError as exc:
        print(f"ERROR: {exc}", file=sys.stderr)
        return 2

    if violations:
        print("VIOLATION: Aliyun Tea RuntimeOptions must go through direct_runtime_options()")
        for item in violations:
            print(item)
        print(
            "\nHelper: task2app/Saas_project/cloud/providers/aliyun/direct_network.py"
            "\nRule: .ai/01_project_constraints/23_app_startup_no_env_proxy.md"
        )
        return 1

    print("OK: no hand-written Aliyun Tea RuntimeOptions() outside allowlist")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
