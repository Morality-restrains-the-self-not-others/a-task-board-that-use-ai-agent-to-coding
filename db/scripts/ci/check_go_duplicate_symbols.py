#!/usr/bin/env python3
"""Detect duplicate package-level func/type names in a Go package dir.

Catches WIP splits that leave the same `func Foo` in two files — go build then
fails with a terse exit 1 that runAll surfaces poorly.

Only scans top-level `func Name(` (no methods) and `type Name`. Skips const/var
iota blocks (regex-unfriendly; the compiler already reports those clearly).

Usage:
  python3 db/scripts/ci/check_go_duplicate_symbols.py path/to/pkg
  python3 db/scripts/ci/check_go_duplicate_symbols.py --src-dir ./src

Exit:
  0 — ok
  1 — duplicates found
  2 — usage / IO error
"""

from __future__ import annotations

import argparse
import re
import sys
from collections import defaultdict
from pathlib import Path

# Package-level only: column-0 `func Name(` (not methods) / `type Name`.
_FUNC = re.compile(r"^func\s+([A-Za-z_]\w*)\s*\(")
_TYPE = re.compile(r"^type\s+([A-Za-z_]\w*)\b")


def _has_build_constraint(path: Path) -> bool:
    """Return True if the file has a //go:build constraint (other than
    //go:build ignore), meaning it is only compiled with specific -tags.
    The checker cannot resolve active tags, so skip these files to avoid
    false-positive duplicate reports."""
    try:
        text = path.read_text(encoding="utf-8", errors="replace")
    except OSError:
        return False
    for line in text.splitlines():
        s = line.strip()
        if not s:
            continue
        if s.startswith("//go:build") and "ignore" not in s:
            return True
        # Stop at first non-empty, non-comment line — build constraints
        # must appear at the top of the file.
        if not s.startswith("//") and not s.startswith("/*"):
            return False
    return False


def scan_file(path: Path) -> list[tuple[str, str, int]]:
    """Return list of (kind, name, line_no) for package-level decls."""
    text = path.read_text(encoding="utf-8", errors="replace")
    out: list[tuple[str, str, int]] = []
    for i, line in enumerate(text.splitlines(), start=1):
        # Skip clearly non-declaration lines early.
        if not line or line[0] in " \t\r#/" or line.startswith("//"):
            continue
        m = _FUNC.match(line)
        if m:
            name = m.group(1)
            if name == "init":
                continue
            out.append(("func", name, i))
            continue
        m = _TYPE.match(line)
        if m:
            out.append(("type", m.group(1), i))
    return out


def check_dir(pkg_dir: Path) -> list[str]:
    if not pkg_dir.is_dir():
        raise FileNotFoundError(f"not a directory: {pkg_dir}")
    by_key: dict[tuple[str, str], list[tuple[Path, int]]] = defaultdict(list)
    for path in sorted(pkg_dir.glob("*.go")):
        if _has_build_constraint(path):
            continue
        try:
            decls = scan_file(path)
        except OSError as exc:
            raise OSError(f"read {path}: {exc}") from exc
        for kind, name, line_no in decls:
            by_key[(kind, name)].append((path, line_no))

    errors: list[str] = []
    for (kind, name), locs in sorted(by_key.items()):
        if len(locs) < 2:
            continue
        detail = ", ".join(f"{p.name}:{ln}" for p, ln in locs)
        errors.append(f"duplicate {kind} {name}: {detail}")
    return errors


def main(argv: list[str] | None = None) -> int:
    ap = argparse.ArgumentParser(description=__doc__)
    ap.add_argument(
        "paths",
        nargs="*",
        help="Go package directories. Default: ./src if present.",
    )
    ap.add_argument(
        "--src-dir",
        action="append",
        default=[],
        help="Extra package dir (repeatable). Convenience for build.sh.",
    )
    args = ap.parse_args(argv)

    dirs: list[Path] = []
    for raw in list(args.paths) + list(args.src_dir):
        p = Path(raw).resolve()
        if p.is_file():
            p = p.parent
        dirs.append(p)
    if not dirs:
        cand = Path("src").resolve()
        if cand.is_dir():
            dirs.append(cand)
        else:
            print("ERROR: pass a Go package directory", file=sys.stderr)
            return 2

    all_errs: list[str] = []
    for d in dirs:
        try:
            errs = check_dir(d)
        except (OSError, FileNotFoundError) as exc:
            print(f"ERROR: {exc}", file=sys.stderr)
            return 2
        if errs:
            all_errs.append(f"[{d}]")
            all_errs.extend(f"  {e}" for e in errs)

    if all_errs:
        print("ERROR: duplicate Go package symbols detected:", file=sys.stderr)
        print("\n".join(all_errs), file=sys.stderr)
        print(
            "Hint: WIP helpers often leave the same func in two files after a split.",
            file=sys.stderr,
        )
        return 1

    for d in dirs:
        print(f"OK duplicate-symbols [{d}]")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
