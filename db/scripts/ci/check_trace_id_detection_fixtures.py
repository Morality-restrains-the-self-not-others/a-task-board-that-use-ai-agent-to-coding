#!/usr/bin/env python3
"""Verify traceId extraction fixtures (case-insensitive) for brainstorming skill §1."""

from __future__ import annotations

import re
import sys
from pathlib import Path

try:
    import yaml
except ImportError:  # pragma: no cover
    print("ERROR: PyYAML required", file=sys.stderr)
    raise SystemExit(2)

HERE = Path(__file__).resolve().parent
FIXTURES = HERE / "trace_id_detection_fixtures.yaml"

# Mirrors .claude/skills/1-brainstorming-design-docs/SKILL.md §1 patterns (case-insensitive keys)
PATTERNS = [
    re.compile(r"(?i)\btrace[_-]?id\b\s*[:=]\s*['\"]?([a-zA-Z0-9_-]{8,})"),
    re.compile(r'(?i)data-trace-?id\s*=\s*["\']([^"\']+)["\']'),
    re.compile(r"(?i)x-trace-id\s*:\s*(\S+)"),
    re.compile(r'(?i)"trace[_-]?id"\s*:\s*"([^"]+)"'),
]


def extract_trace_id(text: str) -> str | None:
    for pat in PATTERNS:
        m = pat.search(text)
        if m:
            return m.group(1)
    return None


def main() -> int:
    data = yaml.safe_load(FIXTURES.read_text(encoding="utf-8")) or {}
    fixtures = data.get("fixtures") or []
    errors: list[str] = []
    for fx in fixtures:
        got = extract_trace_id(fx["input"])
        want = fx["expected"]
        if got != want:
            errors.append(f"{fx['id']}: got {got!r} want {want!r}")

    if errors:
        print("traceId fixture failures:", file=sys.stderr)
        for e in errors:
            print(f"  - {e}", file=sys.stderr)
        return 1

    print(f"OK: {len(fixtures)} traceId fixtures")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
