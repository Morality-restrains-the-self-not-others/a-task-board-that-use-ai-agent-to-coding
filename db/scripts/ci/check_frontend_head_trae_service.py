#!/usr/bin/env python3
"""Ensure entry HTML pages declare <meta name="trae-service" ...>.

SSOT: db/scripts/ci/frontend_head_trae_service.yaml
Rule: .ai/01_project_constraints/26_frontend_head_trae_service.md

Exit codes:
  0 — all listed pages contain the expected meta
  1 — missing / wrong meta
  2 — configuration / IO error
"""

from __future__ import annotations

import argparse
import re
import sys
from pathlib import Path
from typing import Any

try:
    import yaml
except ImportError:  # pragma: no cover
    print("ERROR: PyYAML required (import yaml)", file=sys.stderr)
    raise SystemExit(2)

META_RE = re.compile(
    r"""<meta\s+[^>]*name\s*=\s*["']trae-service["'][^>]*>""",
    re.IGNORECASE,
)
CONTENT_RE = re.compile(
    r"""content\s*=\s*["']([^"']+)["']""",
    re.IGNORECASE,
)


def monorepo_root() -> Path:
    here = Path(__file__).resolve()
    for parent in here.parents:
        if (parent / "db" / "registry.yaml").is_file():
            return parent
    raise FileNotFoundError("db/registry.yaml not found above checker")


def load_entries(path: Path) -> list[dict[str, Any]]:
    data = yaml.safe_load(path.read_text(encoding="utf-8")) or {}
    if not isinstance(data, dict):
        raise ValueError(f"{path}: root must be a mapping")
    entries = data.get("entries")
    if not isinstance(entries, list) or not entries:
        raise ValueError(f"{path}: entries must be a non-empty list")
    return entries


def extract_trae_service(html: str) -> str | None:
    m = META_RE.search(html)
    if not m:
        return None
    tag = m.group(0)
    cm = CONTENT_RE.search(tag)
    if not cm:
        return None
    return cm.group(1).strip()


def check(root: Path, manifest: Path) -> list[str]:
    errors: list[str] = []
    for entry in load_entries(manifest):
        rel = str(entry.get("path") or "").strip()
        expect = str(entry.get("service") or "").strip()
        if not rel or not expect:
            errors.append(f"invalid entry: {entry!r}")
            continue
        path = root / rel
        if not path.is_file():
            errors.append(f"missing file: {rel}")
            continue
        text = path.read_text(encoding="utf-8")
        got = extract_trae_service(text)
        if got is None:
            errors.append(f"{rel}: missing <meta name=\"trae-service\" ...>")
        elif got != expect:
            errors.append(f"{rel}: trae-service={got!r}, expected {expect!r}")
    return errors


def main(argv: list[str] | None = None) -> int:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument(
        "--manifest",
        type=Path,
        default=None,
        help="YAML manifest (default: alongside this script)",
    )
    args = parser.parse_args(argv)
    try:
        root = monorepo_root()
    except FileNotFoundError as e:
        print(f"ERROR: {e}", file=sys.stderr)
        return 2
    manifest = args.manifest or (Path(__file__).resolve().parent / "frontend_head_trae_service.yaml")
    try:
        errors = check(root, manifest)
    except Exception as e:  # noqa: BLE001 — surface config/IO as exit 2
        print(f"ERROR: {e}", file=sys.stderr)
        return 2
    if errors:
        print("FAIL: frontend head trae-service checks:")
        for err in errors:
            print(f"  - {err}")
        return 1
    print("OK: all listed entry HTML pages declare trae-service")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
