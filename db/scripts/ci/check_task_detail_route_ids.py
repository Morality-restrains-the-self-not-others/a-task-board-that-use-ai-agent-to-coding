#!/usr/bin/env python3
"""Forbid hand-written props.* || route.params.* ID assembly in task-detail SSOT paths."""

from __future__ import annotations

import re
import sys
from pathlib import Path

ROOT = Path(__file__).resolve().parents[3]
TARGETS = [
    ROOT / "task2app" / "front_project" / "app" / "src" / "components" / "task-detail",
    ROOT / "task2app" / "front_project" / "app" / "src" / "composables" / "useTaskDetail.js",
    ROOT / "task2app" / "front_project" / "app" / "src" / "components" / "TaskDetailContent.logic.vue",
]
PATTERN = re.compile(
    r"props\.\w+\s*\|\|\s*route\.params\.|route\.params\.\w+\s*\|\|\s*props\.",
)
ALLOW_FILE = ROOT / "task2app" / "front_project" / "app" / "src" / "utils" / "resolveTaskRouteIds.js"


def scan_file(path: Path) -> list[str]:
    if path.resolve() == ALLOW_FILE.resolve():
        return []
    text = path.read_text(encoding="utf-8")
    hits: list[str] = []
    for i, line in enumerate(text.splitlines(), start=1):
        if PATTERN.search(line):
            hits.append(f"{path.relative_to(ROOT)}:{i}: {line.strip()[:120]}")
    return hits


def main() -> int:
    all_hits: list[str] = []
    for target in TARGETS:
        if target.is_dir():
            for path in sorted(target.rglob("*")):
                if path.suffix in {".vue", ".js", ".ts"}:
                    all_hits.extend(scan_file(path))
        elif target.is_file():
            all_hits.extend(scan_file(target))

    if all_hits:
        print("Use resolveTaskRouteIds instead of props||route.params ID assembly:", file=sys.stderr)
        for h in all_hits:
            print(f"  - {h}", file=sys.stderr)
        return 1

    print("OK: no forbidden route id assembly in task-detail paths")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
