#!/usr/bin/env python3
"""Gate: .full.archimate must inherit Views from the previous same-view .full.

Rule / design:
  docs/superpowers/specs/2026-09-04-archimate-full-view-inheritance-design.md
  .claude/skills/1-brainstorming-design-docs/SKILL.md Step 3e-2

Checks (per view name: enterprise-landscape, application-integration, …):
  1. For each .full with version N, locate prior same-view .full (max M < N)
     under docs/architecture/ and docs/architecture/archive/.
  2. Every ArchimateDiagramModel/@name in prior must appear in current
     (Views inheritance).
  3. Current view count must be >= prior view count.
  4. Soft anti-slice: if prior had >= 3 views and a sibling .diff exists with
     the same element+view counts as .full, fail (full≈diff regression).

Known debt: none (OPT-20260904-011 backfilled v126–v132 Views inheritance).
Legacy `--strict-debt` flag kept as a no-op alias for scripts.

Usage:
  python3 db/scripts/ci/check_archimate_full_inherits_views.py
  python3 db/scripts/ci/check_archimate_full_inherits_views.py --root /path
  python3 db/scripts/ci/check_archimate_full_inherits_views.py --strict-debt
"""

from __future__ import annotations

import argparse
import re
import sys
from dataclasses import dataclass
from pathlib import Path

FULL_RE = re.compile(
    r"^v(?P<ver>\d+)-(?P<view>.+)-(?P<ts>\d{8}-\d{4})-(?P<author>.+)\.full\.archimate$"
)
ELEM_RE = re.compile(r'<element\s+xsi:type="archimate:')

# Cleared after OPT-20260904-011 backfill (was range(126, 133)).
DEBT_VERSIONS: frozenset[int] = frozenset()


@dataclass(frozen=True)
class FullFile:
    path: Path
    version: int
    view: str


def parse_full_name(path: Path) -> FullFile | None:
    m = FULL_RE.match(path.name)
    if not m:
        return None
    return FullFile(path=path, version=int(m.group("ver")), view=m.group("view"))


def diagram_names(text: str) -> list[str]:
    found = re.findall(
        r'xsi:type="archimate:ArchimateDiagramModel"[^>]*\bname="([^"]+)"'
        r'|\bname="([^"]+)"[^>]*xsi:type="archimate:ArchimateDiagramModel"',
        text,
    )
    out = [a or b for a, b in found]
    if out:
        return out
    return re.findall(r'ArchimateDiagramModel[^>]*name="([^"]+)"', text)


def element_count(text: str) -> int:
    return len(ELEM_RE.findall(text))


def collect_fulls(arch_root: Path) -> list[FullFile]:
    files: list[FullFile] = []
    for base in (arch_root, arch_root / "archive"):
        if not base.is_dir():
            continue
        for p in base.glob("v*-*.full.archimate"):
            ff = parse_full_name(p)
            if ff:
                files.append(ff)
    return files


def prior_for(target: FullFile, all_fulls: list[FullFile]) -> FullFile | None:
    cands = [
        f
        for f in all_fulls
        if f.view == target.view and f.version < target.version
    ]
    if not cands:
        return None
    return max(cands, key=lambda f: f.version)


def check_pair(
    current: FullFile,
    prior: FullFile,
    *,
    strict_debt: bool,
) -> list[str]:
    cur_text = current.path.read_text(encoding="utf-8")
    pri_text = prior.path.read_text(encoding="utf-8")
    cur_views = diagram_names(cur_text)
    pri_views = diagram_names(pri_text)
    missing = sorted(set(pri_views) - set(cur_views))
    errors: list[str] = []
    in_debt = current.version in DEBT_VERSIONS

    def emit(msg: str) -> None:
        if in_debt and not strict_debt:
            errors.append(f"WARN: {msg} (debt v126–v132; use --strict-debt to fail)")
        else:
            errors.append(msg)

    if missing:
        emit(
            f"{current.path.name}: missing inherited Views from "
            f"{prior.path.name}: {missing}"
        )
    if len(cur_views) < len(pri_views):
        emit(
            f"{current.path.name}: view count {len(cur_views)} < prior "
            f"{prior.path.name} count {len(pri_views)}"
        )

    if len(pri_views) >= 3:
        diff = current.path.with_name(
            current.path.name.replace(".full.archimate", ".diff.archimate")
        )
        if diff.is_file():
            diff_text = diff.read_text(encoding="utf-8")
            if (
                element_count(cur_text) == element_count(diff_text)
                and len(cur_views) == len(diagram_names(diff_text))
            ):
                emit(
                    f"{current.path.name}: full≈diff slice "
                    f"(elements={element_count(cur_text)}, views={len(cur_views)}); "
                    f"prior {prior.path.name} had {len(pri_views)} views"
                )

    return errors


def run(arch_root: Path, *, strict_debt: bool = False) -> int:
    all_fulls = collect_fulls(arch_root)
    targets = [
        f for f in all_fulls if f.path.parent.resolve() == arch_root.resolve()
    ]
    if not targets:
        print("OK: no .full.archimate under docs/architecture/ (nothing to check)")
        return 0

    hard_errors: list[str] = []
    warns: list[str] = []
    checked = 0
    for t in sorted(targets, key=lambda f: (f.view, f.version)):
        prior = prior_for(t, all_fulls)
        if prior is None:
            print(f"SKIP {t.path.name}: no prior same-view .full")
            continue
        checked += 1
        for msg in check_pair(t, prior, strict_debt=strict_debt):
            if msg.startswith("WARN:"):
                warns.append(msg)
            else:
                hard_errors.append(msg)

    for w in warns:
        print(w)
    for e in hard_errors:
        print(f"FAIL: {e}")

    if hard_errors:
        print(
            f"\nVIOLATION: {len(hard_errors)} .full.archimate inheritance failure(s); "
            f"checked={checked}. See docs/superpowers/specs/"
            f"2026-09-04-archimate-full-view-inheritance-design.md"
        )
        return 1

    print(
        f"OK: archimate .full Views inheritance — checked={checked}, "
        f"warns={len(warns)} (debt versions soft)"
    )
    return 0


def main(argv: list[str] | None = None) -> int:
    p = argparse.ArgumentParser(description=__doc__)
    p.add_argument(
        "--root",
        type=Path,
        default=None,
        help="Repo root (default: detect from script location)",
    )
    p.add_argument(
        "--strict-debt",
        action="store_true",
        help="Also fail on known-broken versions v126–v132",
    )
    args = p.parse_args(argv)
    root = args.root
    if root is None:
        root = Path(__file__).resolve().parents[3]
    arch = root / "docs" / "architecture"
    if not arch.is_dir():
        print(f"OK: no {arch} directory")
        return 0
    return run(arch, strict_debt=args.strict_debt)


if __name__ == "__main__":
    raise SystemExit(main())
