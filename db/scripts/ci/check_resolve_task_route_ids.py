#!/usr/bin/env python3
"""Forbid hand-rolled tenant/workspace/task id assembly outside resolveTaskRouteIds."""
from __future__ import annotations
import re
import sys
from pathlib import Path

ROOT = Path(__file__).resolve().parents[3]
APP = ROOT / "task2app/front_project/app/src"
# props.x || route.params.y style
BAD = re.compile(
    r"(?:props|route)\.(?:params\.)?(?:tenantId|tenant_id|workspaceId|workspace_id|taskId|task_id)\s*\|\|"
)
SCOPE_DIRS = [
    APP / "components/task-detail",
    APP / "composables",
]
SCOPE_FILES = [
    APP / "composables/useTaskDetail.js",
    APP / "components/TaskDetailContent.logic.vue",
]


def main() -> int:
    bad = []
    paths = []
    for d in SCOPE_DIRS:
        if d.is_dir():
            paths.extend(d.rglob("*.{js,vue,ts}".replace("{js,vue,ts}", "*")))
    # rglob with brace doesn't work — fix
    paths = []
    for d in SCOPE_DIRS:
        if d.is_dir():
            paths.extend(list(d.rglob("*.js")) + list(d.rglob("*.vue")) + list(d.rglob("*.ts")))
    for f in SCOPE_FILES:
        if f.exists():
            paths.append(f)
    for p in sorted(set(paths)):
        if "resolveTaskRouteIds" in p.name:
            continue
        text = p.read_text(encoding="utf-8", errors="replace")
        for i, line in enumerate(text.splitlines(), 1):
            if BAD.search(line):
                bad.append(f"{p.relative_to(ROOT)}:{i}:{line.strip()[:120]}")
    if bad:
        print("resolveTaskRouteIds hand-roll check FAILED:")
        for b in bad[:40]:
            print(" ", b)
        return 1
    print("ok: no hand-rolled route id assembly in scoped paths")
    return 0


if __name__ == "__main__":
    sys.exit(main())
