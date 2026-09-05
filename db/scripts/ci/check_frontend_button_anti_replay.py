#!/usr/bin/env python3
"""Frontend button anti-replay gate (constraint 52 / ADR-0020).

Default: required artifacts + clickGuard exports exist.
--strict --files: Vue with @click + mutating HTTP must use createClickGuard /
Idempotency-Key / Anti-Replay-OK.

用法:
  check_frontend_button_anti_replay.py
  check_frontend_button_anti_replay.py --strict --files 'taskFE/app/src/views/OrderCreate.vue'
"""
from __future__ import annotations

import argparse
import re
import sys
from pathlib import Path

ROOT = Path(__file__).resolve().parents[3]

REQUIRED_REL = (
    ".ai/01_project_constraints/57_frontend_button_anti_replay.md",
    ".cursor/rules/frontend-button-anti-replay.mdc",
    "docs/adr/0020-frontend-button-anti-replay.md",
    "taskFE/app/src/utils/clickGuard.js",
    ".ai/01_project_constraints/00_project_constraints.md",
)

HELPER_EXPORTS = ("createClickGuard", "mergeIdempotencyHeaders", "IDEMPOTENCY_HEADER")
CLICK_RE = re.compile(r"@click")
MUTATE_RE = re.compile(
    r"method\s*:\s*['\"](?:POST|PUT|PATCH|DELETE)['\"]",
    re.IGNORECASE,
)
GUARD_RE = re.compile(
    r"createClickGuard|Idempotency-Key|idempotencyKey|Anti-Replay-OK"
)


def missing_artifacts(root: Path) -> list[str]:
    hits: list[str] = []
    for rel in REQUIRED_REL:
        p = root / rel
        if not p.is_file():
            hits.append(f"missing {rel}")
    helper = root / "taskFE/app/src/utils/clickGuard.js"
    if helper.is_file():
        text = helper.read_text(encoding="utf-8", errors="replace")
        for name in HELPER_EXPORTS:
            if name not in text:
                hits.append(f"clickGuard.js missing export {name}")
    constraints = root / ".ai/01_project_constraints/00_project_constraints.md"
    if constraints.is_file():
        body = constraints.read_text(encoding="utf-8", errors="replace")
        if "52. **" not in body or "前端按钮" not in body:
            hits.append("00_project_constraints.md missing item 52 前端按钮防重放")
    return hits


def vue_mutating_click_unguarded(text: str) -> bool:
    return bool(CLICK_RE.search(text) and MUTATE_RE.search(text) and not GUARD_RE.search(text))


def resolve_file_patterns(root: Path, files_arg: str) -> list[Path]:
    targets: list[Path] = []
    for pattern in files_arg.split(","):
        pattern = pattern.strip()
        if not pattern:
            continue
        direct = Path(pattern)
        if direct.is_file():
            targets.append(direct.resolve())
            continue
        rel = root / pattern
        if rel.is_file():
            targets.append(rel.resolve())
            continue
        if not direct.is_absolute():
            targets.extend(sorted(root.glob(pattern)))
    return targets


def scan_vue_files(root: Path, files: list[Path]) -> list[str]:
    hits: list[str] = []
    for p in files:
        if p.suffix != ".vue":
            continue
        text = p.read_text(encoding="utf-8", errors="replace")
        if vue_mutating_click_unguarded(text):
            try:
                rel = p.relative_to(root)
            except ValueError:
                rel = p
            hits.append(f"{rel}: mutating @click without createClickGuard / Idempotency-Key / Anti-Replay-OK")
    return hits


def main(argv: list[str] | None = None) -> int:
    ap = argparse.ArgumentParser(description=__doc__)
    ap.add_argument("--strict", action="store_true")
    ap.add_argument("--files", default="", help="comma-separated globs (Vue scan)")
    ap.add_argument("--root", default="", help="override repo root (tests)")
    args = ap.parse_args(argv)
    root = Path(args.root).resolve() if args.root else ROOT

    hits = missing_artifacts(root)
    vue_hits: list[str] = []
    if args.files:
        targets = resolve_file_patterns(root, args.files)
        vue_hits = scan_vue_files(root, targets)
        if args.strict:
            hits.extend(vue_hits)
        else:
            for h in vue_hits:
                print(f"WARN  {h}")

    for h in hits:
        print(f"ERROR {h}")
    if hits:
        print(f"frontend-button-anti-replay: {len(hits)} violation(s)")
        return 1
    scanned = f", scanned {len(vue_hits)} warn" if vue_hits and not args.strict else ""
    print(f"ok: frontend button anti-replay artifacts{scanned}")
    return 0


if __name__ == "__main__":
    sys.exit(main())
