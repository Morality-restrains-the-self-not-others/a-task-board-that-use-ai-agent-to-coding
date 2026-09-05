#!/usr/bin/env python3
"""Bug-fix commits must stage a matching regression unit test.

SSOT: .ai/01_project_constraints/41_bug_fix_unit_test_required.md
Wired: .githooks/commit-msg ($1 = message file; authoritative — commit-msg is the
      only stage where the final message is available: with `git commit -m` the
      pre-commit stage COMMIT_EDITMSG still holds the PREVIOUS commit's message,
      so reading it there causes false blocks) + .pre-commit-config.yaml
      (stages: [commit-msg])

A commit whose message classifies as a bug fix (fix:/hotfix:/bugfix: prefix,
Chinese "修复"/"修正" prefixes, or fix/bug keywords) is BLOCKED unless the
staged changes include a unit test file that corresponds to the modified
source files (same directory / same package / tests mirror / stem overlap).

Exemptions (pass without a test):
  - commit message contains "no-test:" with a stated reason
  - staged changes contain no business source code (docs/config/scripts only)
  - staged changes are unit tests only
  - "revert" / other non-fix conventional prefixes

Exit codes:
  0 — not a bug-fix commit, or exempted, or a matching unit test is staged
  1 — bug-fix commit without a matching unit test (blocked)
  2 — configuration / IO error
"""

from __future__ import annotations

import argparse
import posixpath
import re
import subprocess
import sys
from pathlib import Path

# ── Bug-fix message detection ────────────────────────────────────────────────

# Conventional non-fix prefixes short-circuit keyword detection first.
NON_FIX_PREFIX_RE = re.compile(
    r"^(feat|feature|docs|refactor|refact|test|tests|chore|ci|style|perf|"
    r"build|revert|merge|release|deps?)(\([^)]*\))?[:：]",
    re.IGNORECASE,
)
FIX_PREFIX_RE = re.compile(r"^(fix|hotfix|bugfix|patch)(\([^)]*\))?[:：]", re.IGNORECASE)
FIX_CN_RE = re.compile(r"^(修复|修正|修\s?bug|bug\s?修复|补丁)([:：\s])", re.IGNORECASE)
FIX_KEYWORD_RE = re.compile(r"\b(fix(?:es|ed)?|bug|bugfix|hotfix)\b", re.IGNORECASE)

# ── Staged file classification ───────────────────────────────────────────────

SOURCE_SUFFIXES = (".go", ".py", ".js", ".jsx", ".ts", ".tsx", ".vue", ".mjs", ".cjs")

# Path segments that mark a file as non-unit-test (e2e / fixtures / generated).
NON_UNIT_DIR_SEGMENTS = {
    "node_modules",
    ".git",
    "vendor",
    "venv",
    ".venv",
    "site-packages",
    "__pycache__",
    "playwright",
    "e2e",
    "e2e-tests",
    "end_to_end_tests",
    "testdata",
    "mocks",
}
NON_UNIT_NAME_MARKERS = (".playwright.test.", ".e2e.test.", ".integration.test.", ".spec.e2e.")


def is_unit_test_file(rel: str) -> bool:
    """True if rel names a first-party unit test file (not e2e/playwright)."""
    norm = rel.replace("\\", "/")
    name = posixpath.basename(norm)
    if any(seg in NON_UNIT_DIR_SEGMENTS for seg in norm.split("/")):
        return False
    low = norm.lower()
    if any(marker in low for marker in NON_UNIT_NAME_MARKERS):
        return False
    if name.endswith("_test.go"):
        return True
    if name.endswith(".py") and (name.startswith("test_") or name.endswith("_test.py")):
        return True
    return bool(re.search(r"\.(unit\.)?(test|spec)\.(js|jsx|ts|tsx|mjs|cjs)$", name, re.IGNORECASE))


def is_source_file(rel: str) -> bool:
    """True if rel is business source code (Go/Python/JS/TS/Vue), not a test."""
    if is_unit_test_file(rel):
        return False
    return rel.replace("\\", "/").lower().endswith(SOURCE_SUFFIXES)


def has_matching_test(src: str, test_files: list[str]) -> bool:
    """Regression test corresponds to src: same dir, tests/ mirror, or stem overlap."""
    src_dir = posixpath.dirname(src)
    src_stem = posixpath.basename(src).rsplit(".", 1)[0]
    for t in test_files:
        t_dir = posixpath.dirname(t)
        if t_dir == src_dir:
            return True
        if t_dir == posixpath.join(src_dir, "tests"):
            return True
        if src_stem in posixpath.basename(t):
            return True
    return False


# ── Message detection ────────────────────────────────────────────────────────

def is_bug_fix_message(message: str) -> bool:
    """Classify a commit message's first line as a bug-fix commit."""
    first = (message or "").strip().splitlines()[0] if (message or "").strip() else ""
    if not first:
        return False
    if NON_FIX_PREFIX_RE.match(first):
        return False
    if FIX_PREFIX_RE.match(first):
        return True
    if FIX_CN_RE.match(first):
        return True
    return bool(FIX_KEYWORD_RE.search(first))


# ── Git plumbing ─────────────────────────────────────────────────────────────

def git_output(root: Path, args: list[str]) -> str:
    completed = subprocess.run(
        ["git"] + args,
        cwd=str(root),
        capture_output=True,
        text=True,
        check=False,
    )
    return completed.stdout or ""


def read_staged_files(root: Path) -> list[str]:
    return [
        line.strip()
        for line in git_output(root, ["diff", "--cached", "--name-only", "--diff-filter=ACMR"]).splitlines()
        if line.strip()
    ]


def read_commit_message(root: Path) -> str:
    msg_path = git_output(root, ["rev-parse", "--git-path", "COMMIT_EDITMSG"]).strip()
    if not msg_path:
        msg_path = str(root / ".git" / "COMMIT_EDITMSG")
    if not Path(msg_path).is_absolute():
        msg_path = str(root / msg_path)
    try:
        return Path(msg_path).read_text(encoding="utf-8", errors="replace")
    except OSError:
        return ""


# ── Gate ─────────────────────────────────────────────────────────────────────

VIOLATION_HEADER = (
    "⛔ VIOLATION (rule .ai/01_project_constraints/41_bug_fix_unit_test_required.md): "
    "bug-fix commit without a matching regression unit test."
)


def violation(detail: str, fix_hint: str) -> str:
    return f"{VIOLATION_HEADER}\n{detail}\n{fix_hint}"


def run_gate(root: Path, message: str, staged_files: list[str], verbose: bool, force_bugfix: bool = False) -> int:
    if not staged_files:
        if verbose:
            print("OK: no staged files to check.")
        return 0

    if not force_bugfix and not is_bug_fix_message(message):
        first = (message or "").strip().splitlines()[0] if message.strip() else ""
        if verbose:
            print(f"OK: not a bug-fix commit (first line: {first!r}).")
        return 0

    source_files = [f for f in staged_files if is_source_file(f)]
    test_files = [f for f in staged_files if is_unit_test_file(f)]

    if not source_files:
        print("OK: bug-fix commit touches no business source (docs/config/scripts/tests only) — exempt.")
        return 0

    if "no-test:" in (message or ""):
        reason = ""
        for line in (message or "").splitlines():
            if "no-test:" in line:
                reason = line.split("no-test:", 1)[1].strip()
                break
        print(f"OK: explicit waiver `no-test:` present ({reason or 'reason'}) — exempt.")
        return 0

    if not test_files:
        detail = "  Staged source files: " + ", ".join(source_files)
        hint = (
            "  Fix: add a regression unit test that reproduces the defect (it must fail before "
            "the fix and pass after), then stage it together with the fix.\n"
            "  Waiver: if this is a documented exception, add `no-test: <reason>` to the commit message."
        )
        print(violation(detail, hint))
        return 1

    unmatched = [s for s in source_files if not has_matching_test(s, test_files)]
    if unmatched:
        detail = (
            "  Staged unit tests do not correspond to the modified source:\n"
            "    sources without a matching test: " + ", ".join(unmatched) + "\n"
            "    staged tests: " + ", ".join(test_files)
        )
        hint = (
            "  Fix: add a test in the same directory/package as the modified source "
            "(or a tests/ mirror), e.g. parser.go -> parser_test.go, models.py -> tests/test_models.py."
        )
        print(violation(detail, hint))
        return 1

    print("OK: bug-fix commit carries a matching regression unit test.")
    return 0


def main(argv: list[str] | None = None) -> int:
    parser = argparse.ArgumentParser(
        description="Gate: bug-fix commits must stage a matching regression unit test.",
    )
    parser.add_argument("msgfile", nargs="?", default=None,
                        help="commit message file (positional; git commit-msg hooks pass $1)")
    parser.add_argument("--root", default=None, help="monorepo root (default: auto-detect)")
    parser.add_argument("--commit-msg-file", default=None, help="commit message file (default: .git/COMMIT_EDITMSG)")
    parser.add_argument("--message", default=None, help="commit message text (overrides file)")
    parser.add_argument("--staged-files", default=None, help="newline-separated staged file list (overrides git)")
    parser.add_argument("--force-bugfix", action="store_true", help="treat as bug-fix regardless of message")
    parser.add_argument("--verbose", action="store_true", help="print pass diagnostics")
    args = parser.parse_args(argv)

    try:
        root = Path(args.root).resolve() if args.root else monorepo_root()
    except FileNotFoundError as exc:
        print(f"ERROR: {exc}", file=sys.stderr)
        return 2

    message = args.message
    if message is None:
        msg_file = args.msgfile or args.commit_msg_file
        if msg_file:
            try:
                message = Path(msg_file).read_text(encoding="utf-8", errors="replace")
            except OSError:
                message = ""
        else:
            message = read_commit_message(root)

    staged_files = (
        [f.strip() for f in args.staged_files.splitlines() if f.strip()]
        if args.staged_files is not None
        else read_staged_files(root)
    )

    try:
        return run_gate(root, message, staged_files, args.verbose, args.force_bugfix)
    except Exception as exc:  # noqa: BLE001 — fail closed on the commit gate
        print(f"ERROR: gate crashed ({exc!r}); failing closed.", file=sys.stderr)
        return 2


def monorepo_root() -> Path:
    here = Path(__file__).resolve()
    for parent in here.parents:
        if (parent / "db" / "registry.yaml").is_file():
            return parent
    raise FileNotFoundError("db/registry.yaml not found above checker")


if __name__ == "__main__":
    raise SystemExit(main())
