#!/usr/bin/env python3
"""Move an OPT item from open list to completed archive.

Also provides cross-file ID scanning to prevent numbering collisions.
"""
from __future__ import annotations

import re
import sys
from datetime import datetime, timedelta, timezone
from pathlib import Path

TZ = timezone(timedelta(hours=8))
LEARNINGS = Path(__file__).resolve().parent
OPEN = LEARNINGS / "OPTIMIZATION_TODOS.md"
ARCH = LEARNINGS / "OPTIMIZATION_TODOS_COMPLETED.md"
PRODUCT = LEARNINGS / "PRODUCT_DECISIONS.md"
ARCHIVE_DIR = LEARNINGS / "archive" / "completed"

# ── cross-file ID scanning ──────────────────────────────────────────

def _all_opt_files():
    """Yield (label, path) for every file that may contain OPT-* IDs."""
    yield "OPTIMIZATION_TODOS.md", OPEN
    yield "OPTIMIZATION_TODOS_COMPLETED.md", ARCH
    yield "PRODUCT_DECISIONS.md", PRODUCT
    for p in sorted(LEARNINGS.glob("BLOCK_TODO_*.md")):
        yield p.name, p
    if ARCHIVE_DIR.is_dir():
        for p in sorted(ARCHIVE_DIR.glob("OPT_COMPLETED_*.md")):
            yield f"archive/completed/{p.name}", p


def scan_all_opt_ids():
    """Return a set of every OPT-* ID found across all OPT files.

    Matches both standard ``OPT-YYYYMMDD-NNN`` and suffixed variants
    (``-b``, ``-dup``, ``-alt``, etc.).
    """
    ids: set[str] = set()
    id_re = re.compile(r"\bOPT-\d{8}-\d+(?:-[a-z][a-z0-9]*)?\b")
    for label, path in _all_opt_files():
        if not path.exists():
            continue
        try:
            text = path.read_text()
        except Exception:
            continue
        for m in id_re.finditer(text):
            ids.add(m.group(0))
    return ids


def next_available_id(date_str: str | None = None):
    """Return the next available OPT ID for *date_str* (default today).

    Scans **all** OPT files (pending, completed, archive, product-decisions)
    and picks ``max(existing) + 1`` for the given date.
    """
    if date_str is None:
        date_str = datetime.now(TZ).strftime("%Y%m%d")
    prefix = f"OPT-{date_str}-"
    existing = scan_all_opt_ids()
    max_nnn = 0
    for oid in existing:
        if oid.startswith(prefix):
            # Extract the NNN part (handle suffixed variants)
            base = oid[len(prefix):]
            m = re.match(r"(\d{3})", base)
            if m:
                max_nnn = max(max_nnn, int(m.group(1)))
    return f"{prefix}{max_nnn + 1:03d}"


def check_id_availability(opt_id: str):
    """Return (available: bool, conflicts: list[str]).

    *available* is False when *opt_id* (or its base form) already exists.
    *conflicts* lists the files containing the collision.
    """
    existing = scan_all_opt_ids()
    # Exact match
    if opt_id in existing:
        conflicts = []
        id_re = re.compile(r"\b" + re.escape(opt_id) + r"\b")
        for label, path in _all_opt_files():
            if path.exists() and id_re.search(path.read_text()):
                conflicts.append(label)
        return False, conflicts

    # Check if the base ID (without suffix) would be ambiguous
    base = re.sub(r"-[a-z][a-z0-9]*$", "", opt_id)
    if base != opt_id and base in existing:
        conflicts = []
        id_re = re.compile(r"\b" + re.escape(base) + r"\b")
        for label, path in _all_opt_files():
            if path.exists() and id_re.search(path.read_text()):
                conflicts.append(label)
        return False, conflicts

    return True, []


def print_id_report():
    """Print a summary of ID usage across all files (for auditing)."""
    all_ids = sorted(scan_all_opt_ids())
    by_date: dict[str, list[str]] = {}
    for oid in all_ids:
        m = re.match(r"OPT-(\d{8})-", oid)
        date = m.group(1) if m else "unknown"
        by_date.setdefault(date, []).append(oid)

    print(f"Total unique OPT IDs across all files: {len(all_ids)}")
    for date in sorted(by_date.keys()):
        ids = by_date[date]
        # Check for gaps or duplicates in the NNN sequence
        nums = []
        for oid in ids:
            m = re.search(r"OPT-\d{8}-(\d{3})", oid)
            if m:
                nums.append(int(m.group(1)))
        nums.sort()
        gaps = []
        for i in range(1, len(nums)):
            if nums[i] != nums[i-1] + 1:
                for g in range(nums[i-1] + 1, nums[i]):
                    gaps.append(g)
        gap_str = f"  gaps: {gaps}" if gaps else ""
        print(f"  {date}: {len(ids)} IDs (max={max(nums) if nums else 0}){gap_str}")


# ── move ────────────────────────────────────────────────────────────

# Heading formats supported in the pending file:
#   ### OPT-YYYYMMDD-NNN: description          ← actual format
#   ## [OPT-YYYYMMDD-NNN] pending              ← legacy format
#
# Both are detected. The completed archive always uses:
#   ## [OPT-YYYYMMDD-NNN] completed/cancelled

_PENDING_H3_RE = re.compile(
    r"^### (?P<id>OPT-\d{8}-\d+)\s*[:—][^\n]*\n(?P<body>.*?)(?=^### OPT-|\Z)",
    re.M | re.S,
)
_PENDING_H2_RE = re.compile(
    r"^## \[(?P<id>OPT-\d{8}-\d+)\] pending\n(?P<body>.*?)(?=^## \[OPT-|\Z)",
    re.M | re.S,
)
_PENDING_H2_PLAIN_RE = re.compile(
    r"^## (?P<id>OPT-\d{8}-\d+)\s*[:—][^\n]*\n(?P<body>.*?)(?=^## OPT-|\Z)",
    re.M | re.S,
)


# Status 字段兼容：`- **Status**: \`pending\``（当前格式）与 `**Status**: pending`（旧格式）
_STATUS_PENDING_VARIANTS = ("**Status**: pending", "**Status**: `pending`")
_STATUS_COMPLETED_VARIANTS = ("**Status**: completed", "**Status**: `completed`")


def _block_status_ok(body: str) -> bool:
    return any(s in body for s in (_STATUS_PENDING_VARIANTS + _STATUS_COMPLETED_VARIANTS))


def _find_pending_block(text: str, opt_id: str):
    """Return (match_obj, heading_kind) or (None, None).

    heading_kind is ``'h3'`` (### OPT-ID: desc) or ``'h2'`` (## [OPT-ID] pending).
    """
    # Try h3 format first (current standard) — match pending OR completed items not yet migrated
    for m in _PENDING_H3_RE.finditer(text):
        if m.group("id") == opt_id and _block_status_ok(m.group("body")):
            return m, "h3"
    # Fall back to h2 format with brackets (legacy)
    for m in _PENDING_H2_RE.finditer(text):
        if m.group("id") == opt_id:
            return m, "h2"
    # Fall back to plain h2 format (## OPT-ID — desc) — match pending or completed
    for m in _PENDING_H2_PLAIN_RE.finditer(text):
        if m.group("id") == opt_id and _block_status_ok(m.group("body")):
            return m, "h2plain"
    return None, None


def move_opt(opt_id: str, status: str, note: str, completed: str | None = None) -> bool:
    assert status in ("completed", "cancelled")
    text = OPEN.read_text()

    m, kind = _find_pending_block(text, opt_id)
    if m is None:
        print(f"NOT FOUND pending: {opt_id}")
        return False

    # Extract the block (heading + body + trailing separator if present)
    raw_block = m.group(0).rstrip() + "\n\n"

    # Convert heading to archive format
    if kind == "h3":
        # ### OPT-ID: description → ## [OPT-ID] completed
        # Handle both colon (:) and em dash (—) separators
        block = re.sub(
            rf"^### {re.escape(opt_id)}\s*[:—][^\n]*",
            rf"## [{opt_id}] {status}",
            raw_block,
            count=1,
        )
    elif kind == "h2plain":
        # ## OPT-ID — description → ## [OPT-ID] completed
        block = re.sub(
            rf"^## {re.escape(opt_id)}\s*[:—][^\n]*",
            rf"## [{opt_id}] {status}",
            raw_block,
            count=1,
        )
    else:
        # ## [OPT-ID] pending → ## [OPT-ID] completed
        block = re.sub(
            rf"^## \[{re.escape(opt_id)}\] pending",
            rf"## [{opt_id}] {status}",
            raw_block,
            count=1,
        )

    # Update status field（兼容 `- ` 前缀与 backtick 变体，保留前缀）
    block = re.sub(
        r"^(?P<prefix>-\s*)?\*\*Status\*\*: `?(?:pending|completed|cancelled)`?",
        lambda m: f"{m.group('prefix') or ''}**Status**: {status}",
        block,
        count=1,
        flags=re.M,
    )

    completed_ts = completed or datetime.now(TZ).strftime("%Y-%m-%d")
    if "**Completed**:" not in block:
        # 归档格式与 OPTIMIZATION_TODOS_COMPLETED.md 现行约定一致：
        # `- **Completed**: YYYY-MM-DD` + `- **Summary**: <note>`
        block = re.sub(
            r"^((?:-\s*)?\*\*Status\*\*: `?(?:completed|cancelled)`?\n)",
            rf"\1- **Completed**: {completed_ts}\n- **Summary**: {note}\n",
            block,
            count=1,
            flags=re.M,
        )

    # Remove from pending file
    OPEN.write_text(text[: m.start()] + text[m.end() :])

    # Append to completed archive
    arch = ARCH.read_text()
    marker = "## 归档项（completed / cancelled）\n\n"
    if marker in arch:
        arch = arch.replace(marker, marker + block, 1)
    else:
        arch = arch.rstrip() + "\n\n" + block
    ARCH.write_text(arch)
    print(f"MOVED {opt_id} -> {status}")
    return True


# ── CLI ─────────────────────────────────────────────────────────────

def _usage():
    print(__doc__)
    print("Usage:")
    print("  python _move_opt.py <OPT-ID> completed|cancelled '<note>'")
    print("  python _move_opt.py --next-id [YYYYMMDD]")
    print("  python _move_opt.py --check-id <OPT-ID>")
    print("  python _move_opt.py --report")
    sys.exit(1)


if __name__ == "__main__":
    if len(sys.argv) < 2:
        _usage()

    cmd = sys.argv[1]

    if cmd == "--next-id":
        date_str = sys.argv[2] if len(sys.argv) > 2 else None
        nid = next_available_id(date_str)
        print(nid)

    elif cmd == "--check-id":
        if len(sys.argv) < 3:
            _usage()
        opt_id = sys.argv[2]
        avail, conflicts = check_id_availability(opt_id)
        if avail:
            print(f"AVAILABLE: {opt_id}")
        else:
            print(f"CONFLICT: {opt_id} already used in: {', '.join(conflicts)}")
            # Suggest alternatives
            base = re.sub(r"-[a-z][a-z0-9]*$", "", opt_id)
            date_match = re.match(r"(OPT-\d{8})-", base)
            if date_match:
                prefix = date_match.group(1)
                suggested = next_available_id(prefix[4:])  # YYYYMMDD part
                # If the suggested equals the base, bump suffix
                existing = scan_all_opt_ids()
                if suggested not in existing:
                    print(f"  Suggest: {suggested}")
                else:
                    for suffix in ['-b', '-c', '-d', '-e']:
                        candidate = f"{base}-{suffix}" if '-' not in base[len(prefix):] else re.sub(r"-\w+$", f"-{suffix}", base)
                        # Simpler: just try base + suffix
                        candidate = f"{prefix}-{base[len(prefix)+1:]}-{suffix}"
                        if candidate not in existing:
                            print(f"  Suggest: {candidate}")
                            break

    elif cmd == "--report":
        print_id_report()

    else:
        # Legacy: move_opt <id> <status> <note>
        if len(sys.argv) < 4:
            _usage()
        move_opt(sys.argv[1], sys.argv[2], sys.argv[3])
