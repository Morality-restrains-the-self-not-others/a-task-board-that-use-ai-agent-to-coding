#!/usr/bin/env python3
"""Gate: project DESIGN.md + design-md skill stay the visual SSOT.

Ensures:
  - repo-root DESIGN.md exists with Stitch-style required headings
  - YAML frontmatter colors.primary matches taskFE --color-primary
  - .claude/skills/design-md/SKILL.md exists and skills README indexes it

Usage:
  python3 db/scripts/ci/check_design_md.py
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

ROOT = Path(__file__).resolve().parents[3]

CSS_REL = Path("taskFE/app/src/css/tailwind.css")
DESIGN_REL = Path("DESIGN.md")
SKILL_REL = Path(".claude/skills/design-md/SKILL.md")
SKILLS_README_REL = Path(".claude/skills/README.md")

PRIMARY_CSS_RE = re.compile(
    r"--color-primary:\s*(#[0-9A-Fa-f]{3,8})",
    re.IGNORECASE,
)

REQUIRED_HEADINGS = (
    ("Colors", re.compile(r"^##+\s+.*\bcolors?\b", re.IGNORECASE | re.MULTILINE)),
    ("Typography", re.compile(r"^##+\s+.*\btypography\b", re.IGNORECASE | re.MULTILINE)),
    ("Components", re.compile(r"^##+\s+.*\bcomponents?\b", re.IGNORECASE | re.MULTILINE)),
    ("Layout", re.compile(r"^##+\s+.*\blayout\b", re.IGNORECASE | re.MULTILINE)),
    ("Do's and Don'ts", re.compile(r"^##+\s+.*do['’]?s?\s+and\s+don['’]?ts", re.IGNORECASE | re.MULTILINE)),
)


def parse_frontmatter(text: str) -> dict:
    if not text.startswith("---"):
        return {}
    parts = text.split("---", 2)
    if len(parts) < 3:
        return {}
    data = yaml.safe_load(parts[1]) or {}
    return data if isinstance(data, dict) else {}


def extract_css_primary(css_text: str) -> str | None:
    match = PRIMARY_CSS_RE.search(css_text)
    return match.group(1) if match else None


def normalize_hex(value: str) -> str:
    return value.strip().lower()


def collect_violations(root: Path) -> list[str]:
    hits: list[str] = []
    design_path = root / DESIGN_REL
    if not design_path.is_file():
        hits.append(f"{DESIGN_REL}: missing (project visual SSOT required)")
        return hits

    text = design_path.read_text(encoding="utf-8")
    fm = parse_frontmatter(text)
    colors = fm.get("colors") if isinstance(fm.get("colors"), dict) else {}
    primary = colors.get("primary")
    if not isinstance(primary, str) or not primary.strip():
        hits.append(f"{DESIGN_REL}: YAML frontmatter must set colors.primary")
        primary_hex = None
    else:
        primary_hex = primary.strip()

    for label, pattern in REQUIRED_HEADINGS:
        if not pattern.search(text):
            hits.append(f"{DESIGN_REL}: missing required heading {label!r}")

    css_path = root / CSS_REL
    if not css_path.is_file():
        hits.append(f"{CSS_REL}: missing (cannot verify colors.primary)")
    else:
        css_primary = extract_css_primary(css_path.read_text(encoding="utf-8"))
        if not css_primary:
            hits.append(f"{CSS_REL}: missing --color-primary")
        elif primary_hex and normalize_hex(primary_hex) != normalize_hex(css_primary):
            hits.append(
                f"{DESIGN_REL}: colors.primary {primary_hex!r} mismatch "
                f"with {CSS_REL} --color-primary {css_primary!r}"
            )

    skill_path = root / SKILL_REL
    if not skill_path.is_file():
        hits.append(f"{SKILL_REL}: missing (design-md skill required)")

    readme_path = root / SKILLS_README_REL
    if not readme_path.is_file():
        hits.append(f"{SKILLS_README_REL}: missing")
    elif "design-md" not in readme_path.read_text(encoding="utf-8"):
        hits.append(f"{SKILLS_README_REL}: must index `design-md`")

    return hits


def main() -> int:
    hits = collect_violations(ROOT)
    if hits:
        print("DESIGN.md / design-md skill violations:", file=sys.stderr)
        for hit in hits:
            print(f"  - {hit}", file=sys.stderr)
        return 1
    print("OK: DESIGN.md + design-md skill")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
