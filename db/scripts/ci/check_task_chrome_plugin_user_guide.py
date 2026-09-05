#!/usr/bin/env python3
"""Ensure taskChromePlugin USER_GUIDE.md documents every user-guide.js SECTIONS[].id."""

from __future__ import annotations

import re
import sys
from pathlib import Path

ROOT = Path(__file__).resolve().parents[3]
GUIDE_JS = ROOT / "taskChromePlugin" / "lib" / "user-guide.js"
GUIDE_MD = ROOT / "taskChromePlugin" / "docs" / "USER_GUIDE.md"
SECTION_ID_RE = re.compile(r"\bid:\s*['\"]([^'\"]+)['\"]")


def extract_section_ids(js_text: str) -> list[str]:
    # Parse SECTIONS array ids (SSOT in user-guide.js)
    ids: list[str] = []
    in_sections = False
    for line in js_text.splitlines():
        if "const SECTIONS" in line:
            in_sections = True
        if in_sections:
            m = SECTION_ID_RE.search(line)
            if m:
                ids.append(m.group(1))
            if in_sections and line.strip().startswith("];"):
                break
    return ids


def main() -> int:
    if not GUIDE_JS.is_file() or not GUIDE_MD.is_file():
        print("ERROR: user-guide.js or USER_GUIDE.md missing", file=sys.stderr)
        return 2

    ids = extract_section_ids(GUIDE_JS.read_text(encoding="utf-8"))
    md = GUIDE_MD.read_text(encoding="utf-8")
    missing = [i for i in ids if f"`{i}`" not in md and i not in md]

    if missing:
        print("USER_GUIDE.md missing section ids:", file=sys.stderr)
        for mid in missing:
            print(f"  - {mid}", file=sys.stderr)
        return 1

    print(f"OK: {len(ids)} section ids documented")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
