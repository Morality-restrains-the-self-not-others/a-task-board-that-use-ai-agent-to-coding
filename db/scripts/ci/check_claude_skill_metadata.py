#!/usr/bin/env python3
"""Validate .claude/skills/*/SKILL.md frontmatter naming for Cursor discovery.

Rules:
  - name: only [a-z0-9-], non-empty
  - name must equal parent directory basename
  - description: non-empty

Exit 0 pass, 1 violations, 2 config/IO error.
"""

from __future__ import annotations

import re
import sys
from pathlib import Path

try:
    import yaml
except ImportError:  # pragma: no cover
    print("ERROR: PyYAML required", file=sys.stderr)
    raise SystemExit(2)

NAME_RE = re.compile(r"^[a-z0-9-]+$")


def monorepo_root() -> Path:
    here = Path(__file__).resolve()
    for parent in here.parents:
        if (parent / "db" / "registry.yaml").is_file():
            return parent
    raise FileNotFoundError("db/registry.yaml not found")


def parse_frontmatter(text: str) -> dict:
    if not text.startswith("---"):
        return {}
    parts = text.split("---", 2)
    if len(parts) < 3:
        return {}
    data = yaml.safe_load(parts[1]) or {}
    return data if isinstance(data, dict) else {}


def check_skill(skill_md: Path) -> list[str]:
    errors: list[str] = []
    rel = skill_md.parent.relative_to(skill_md.parents[2]) if len(skill_md.parents) > 2 else skill_md.parent
    dir_name = skill_md.parent.name
    fm = parse_frontmatter(skill_md.read_text(encoding="utf-8"))
    name = fm.get("name")
    desc = fm.get("description")

    if not isinstance(name, str) or not name.strip():
        errors.append(f"{rel}: missing or empty name")
    else:
        name = name.strip()
        if not NAME_RE.match(name):
            errors.append(f"{rel}: name {name!r} must match [a-z0-9-]")
        if name != dir_name:
            errors.append(f"{rel}: name {name!r} != directory {dir_name!r}")

    if not isinstance(desc, str) or not desc.strip():
        errors.append(f"{rel}: missing or empty description")

    return errors


def main() -> int:
    root = monorepo_root()
    skills_root = root / ".claude" / "skills"
    if not skills_root.is_dir():
        print(f"ERROR: {skills_root} not found", file=sys.stderr)
        return 2

    all_errors: list[str] = []
    for skill_md in sorted(skills_root.glob("*/SKILL.md")):
        all_errors.extend(check_skill(skill_md))

    if all_errors:
        print("Claude skill metadata violations:", file=sys.stderr)
        for err in all_errors:
            print(f"  - {err}", file=sys.stderr)
        return 1

    print(f"OK: {len(list(skills_root.glob('*/SKILL.md')))} skills checked")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
