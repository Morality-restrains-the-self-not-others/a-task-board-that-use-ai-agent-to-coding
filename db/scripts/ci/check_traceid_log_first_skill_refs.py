#!/usr/bin/env python3
"""Golden：排障类 SKILL.md 须交叉引用 traceid-log-first-diagnosis（或等价措辞）。"""

from __future__ import annotations

import re
import sys
from pathlib import Path

ROOT = Path(__file__).resolve().parents[3]
SKILLS = ROOT / ".claude" / "skills"

# OPT-20260720-010 建议范围
REQUIRED_SKILLS = (
    "pua",
    "logging-audit",
    "webapp-testing",
    "8-build",
    "9-review",
    "1-brainstorming-design-docs",
)

REF_PATTERNS = (
    re.compile(r"traceid-log-first-diagnosis", re.I),
    re.compile(r"有\s*data-traceId\s*先查", re.I),
    re.compile(r"data-traceId.*先.*(日志|Loki|Grafana)", re.I | re.S),
)


def skill_ok(text: str) -> bool:
    return any(p.search(text) for p in REF_PATTERNS)


def main() -> int:
    missing: list[str] = []
    for name in REQUIRED_SKILLS:
        path = SKILLS / name / "SKILL.md"
        if not path.is_file():
            missing.append(f"{name}: missing SKILL.md")
            continue
        text = path.read_text(encoding="utf-8")
        if not skill_ok(text):
            missing.append(
                f"{name}: SKILL.md must reference "
                "traceid-log-first-diagnosis (or equivalent '有 data-traceId 先查日志')"
            )
    if missing:
        print("FAIL check_traceid_log_first_skill_refs:", file=sys.stderr)
        for line in missing:
            print(f"  - {line}", file=sys.stderr)
        return 1
    print(f"OK check_traceid_log_first_skill_refs ({len(REQUIRED_SKILLS)} skills)")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
