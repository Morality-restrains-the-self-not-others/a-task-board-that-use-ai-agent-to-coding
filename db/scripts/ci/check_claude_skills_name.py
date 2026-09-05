#!/usr/bin/env python3
"""Ensure .claude/skills/*/SKILL.md name matches dir and [a-z0-9-]+ with non-empty description."""
from __future__ import annotations
import re
import sys
from pathlib import Path

ROOT = Path(__file__).resolve().parents[3]
SKILLS = ROOT / ".claude" / "skills"
NAME_RE = re.compile(r"^[a-z0-9-]+$")
FM_NAME = re.compile(r"^name:\s*[\"']?([^\"'\n]+)[\"']?\s*$", re.M)
FM_DESC = re.compile(r"^description:\s*(.+)$", re.M)


def main() -> int:
    bad = []
    for skill_md in sorted(SKILLS.glob("*/SKILL.md")):
        dirname = skill_md.parent.name
        text = skill_md.read_text(encoding="utf-8")
        m = FM_NAME.search(text)
        d = FM_DESC.search(text)
        name = (m.group(1).strip() if m else "")
        desc = (d.group(1).strip() if d else "")
        if not name or not NAME_RE.match(name) or name != dirname:
            bad.append(f"{skill_md.relative_to(ROOT)}: name={name!r} dir={dirname!r}")
        if not desc or desc in ("TODO", "placeholder"):
            bad.append(f"{skill_md.relative_to(ROOT)}: empty/placeholder description")
    if bad:
        print("claude skills name check FAILED:")
        for b in bad:
            print(" ", b)
        return 1
    print(f"ok: {len(list(SKILLS.glob('*/SKILL.md')))} skills")
    return 0


if __name__ == "__main__":
    sys.exit(main())
