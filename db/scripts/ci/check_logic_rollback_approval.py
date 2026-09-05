#!/usr/bin/env python3
"""Logic-rollback approval gate (constraint 53 / ADR-0021).

Default: required rule artifacts exist.
With a staged (or injected) source diff: high-signal rollback markers without
`Logic-Rollback-OK:` in the commit message are blocked.

用法:
  python3 db/scripts/ci/check_logic_rollback_approval.py
  python3 db/scripts/ci/check_logic_rollback_approval.py --commit-msg-file .git/COMMIT_EDITMSG
"""
from __future__ import annotations

import argparse
import os
import re
import subprocess
from pathlib import Path

SOURCE_SUFFIXES = (".go", ".py", ".js", ".jsx", ".ts", ".tsx", ".vue", ".sql", ".sh")

REQUIRED_REL = (
    ".ai/01_project_constraints/58_logic_rollback_requires_approval.md",
    ".cursor/rules/logic-rollback-requires-approval.mdc",
    "docs/adr/0021-logic-rollback-requires-approval.md",
    ".ai/01_project_constraints/00_project_constraints.md",
)

IGNORE_REL = {
    "db/scripts/ci/check_logic_rollback_approval.py",
    "db/scripts/ci/test_check_logic_rollback_approval.py",
}

# High-signal: keep/restore superseded behavior. Avoid bare "legacy" (GitLab etc.).
ROLLBACK_LINE_RE = re.compile(
    r"("
    r"keep[_\s-]*old|"
    r"fallback[_\s-]*to[_\s-]*old|"
    r"compat[_\s-]*shim|"
    r"use[_\s-]*old[_\s-]*(path|impl|logic)|"
    r"old[_\s-]*implementation|"
    r"dont[_\s-]*delete|"
    r"do not delete|"
    r"just in case|"
    r"保留旧|"
    r"先留着|"
    r"旧逻辑|"
    r"旧实现|"
    r"回退到|"
    r"兼容旧|"
    r"legacy[_\s-]*(path|impl|handler)|"
    r"keep for compatibility|"
    r"deprecated but keep"
    r")",
    re.IGNORECASE,
)

LEGACY_NAME_RE = re.compile(
    r"(^|/)[\w.-]*(_old|_legacy|_compat|_deprecated|_rollback)[\w.-]*"
    r"\.(go|py|js|jsx|ts|tsx|vue)$",
    re.IGNORECASE,
)

WAIVER_RE = re.compile(r"Logic-Rollback-OK:\s*\S+")
REVERT_MSG_RE = re.compile(
    r"^(revert|rollback|回退)(\([^)]*\))?[:：\s]",
    re.IGNORECASE,
)
COMMENT_LINE_RE = re.compile(r"^\s*(//|#|--|\*|/\*)")
CODEISH_RE = re.compile(
    r"\b(func|def|class|return|if|for|import|export|const|let|var|package)\b"
)
COMMENTED_CODE_MIN = 6

# OPT-20260819-043: 无关键字逻辑回退检测 — 暂存新增的符号若近期在 git 历史中
# 被删除、当前 HEAD 又无同名符号，视为「静默拷回旧逻辑」，同样要求
# Logic-Rollback-OK。避免 Agent 从 git 历史原样拷回已删函数但改名/加注释
# 绕过词表。
DECL_SYMBOL_RE = re.compile(
    r"^\s*(?:"
    r"(?:(?:export\s+default\s+)?(?:async\s+)?function\s+([A-Za-z_$][\w$]*)\s*\()|"
    r"(?:func\s+(?:\([^)]*\)\s+)?([A-Za-z_$][\w$]*)\s*\()|"
    r"(?:def\s+([A-Za-z_$][\w$]*)\s*\()|"
    r"(?:class\s+([A-Za-z_$][\w$]*))"
    r")",
    re.IGNORECASE,
)

Hit = tuple[str, str]  # (path, reason)


def missing_artifacts(root: Path) -> list[str]:
    hits: list[str] = []
    for rel in REQUIRED_REL:
        if not (root / rel).is_file():
            hits.append(f"missing {rel}")
    constraints = root / ".ai/01_project_constraints/00_project_constraints.md"
    if constraints.is_file():
        body = constraints.read_text(encoding="utf-8", errors="replace")
        if "53. **" not in body or "逻辑回退" not in body:
            hits.append("00_project_constraints.md missing item 53 逻辑回退")
    return hits


def is_source_rel(rel: str) -> bool:
    norm = rel.replace("\\", "/")
    if norm in IGNORE_REL:
        return False
    return any(norm.endswith(suf) for suf in SOURCE_SUFFIXES)


def parse_unified_added(diff_text: str) -> dict[str, list[str]]:
    """Map path -> added line texts (without leading '+') from a unified diff."""
    current = ""
    added: dict[str, list[str]] = {}
    for raw in diff_text.splitlines():
        if raw.startswith("+++ "):
            rest = raw[4:].strip().removeprefix("b/")
            current = rest
            added.setdefault(current, [])
            continue
        if raw.startswith("+") and not raw.startswith("+++") and current:
            added.setdefault(current, []).append(raw[1:])
    return added


def scan_added_lines(added: dict[str, list[str]]) -> list[Hit]:
    hits: list[Hit] = []
    for path, lines in added.items():
        if not is_source_rel(path):
            continue
        if LEGACY_NAME_RE.search(path.replace("\\", "/")):
            hits.append((path, "legacy/compat filename"))
        run = 0
        for line in lines:
            if ROLLBACK_LINE_RE.search(line) and "Logic-Rollback-OK:" not in line:
                hits.append((path, f"rollback marker: {line.strip()[:80]}"))
            if COMMENT_LINE_RE.match(line) and CODEISH_RE.search(line):
                run += 1
                if run == COMMENTED_CODE_MIN:
                    hits.append((path, "commented-out code block (>=6 lines)"))
            else:
                run = 0
    return hits


def scan_revert_message(message: str, staged_source: list[str]) -> list[Hit]:
    first = (message or "").lstrip().splitlines()[0] if message else ""
    if not first or not REVERT_MSG_RE.match(first):
        return []
    sources = [p for p in staged_source if is_source_rel(p)]
    if not sources:
        return []
    return [("COMMIT_MESSAGE", f"revert/rollback commit touches source: {sources[0]}")]


def extract_added_symbols(added: dict[str, list[str]]) -> dict[str, list[str]]:
    """Path -> symbols newly declared in added lines (function/def/class)."""
    out: dict[str, list[str]] = {}
    for path, lines in added.items():
        if not is_source_rel(path):
            continue
        syms: list[str] = []
        for line in lines:
            m = DECL_SYMBOL_RE.search(line)
            if not m:
                continue
            sym = next((g for g in m.groups() if g), None)
            if sym and sym not in syms:
                syms.append(sym)
        if syms:
            out[path] = syms
    return out


def _run_git(root: Path, *args: str, timeout: int = 10) -> subprocess.CompletedProcess[str]:
    """Run git against `root` without inheriting hook GIT_DIR/GIT_INDEX_FILE.

    commit-msg hooks inject GIT_* pointing at the host repo; tests (and
    nested git -C) must inspect `root` itself or they rewrite core.bare /
    look at the wrong history.
    """
    env = {k: v for k, v in os.environ.items() if not k.startswith("GIT_")}
    return subprocess.run(
        ["git", "-C", str(root), *args],
        env=env,
        capture_output=True,
        text=True,
        check=False,
        timeout=timeout,
    )


def _symbol_in_ref(root: Path, path: str, ref: str, symbol: str) -> bool:
    """Whether the symbol (as a word) exists in a given git ref of the file."""
    proc = _run_git(root, "show", f"{ref}:{path}", timeout=10)
    if proc.returncode != 0:
        return False
    return re.search(r"\b" + re.escape(symbol) + r"\b", proc.stdout) is not None


def _symbol_in_recent_history(root: Path, path: str, symbol: str, depth: int) -> bool:
    """Whether the symbol appeared in HEAD's ancestry (recent N commits)."""
    proc = _run_git(
        root,
        "log",
        "--format=%h",
        "--max-count",
        str(depth),
        "-S",
        symbol,
        "--",
        path,
        timeout=30,
    )
    return proc.returncode == 0 and bool(proc.stdout.strip())


def scan_history_rollback(
    root: Path,
    added: dict[str, list[str]],
    *,
    history_depth: int = 200,
) -> list[Hit]:
    """Detect symbols re-added from git history after a prior deletion.

    For each newly-added declaration that is absent in HEAD, check whether it
    appeared in recent commits touching that file. If yes, the code is likely a
    silent restore of deleted logic (no keyword marker) and requires approval.
    """
    hits: list[Hit] = []
    for path, syms in extract_added_symbols(added).items():
        for sym in syms:
            # Present in HEAD -> modification/move, not a silent re-add.
            if _symbol_in_ref(root, path, "HEAD", sym):
                continue
            if _symbol_in_recent_history(root, path, sym, history_depth):
                hits.append(
                    (path, f"silent rollback: {sym} existed in recent git history, was removed, re-added")
                )
    return hits


def has_waiver(message: str) -> bool:
    return bool(WAIVER_RE.search(message or ""))


def git_cached_diff(root: Path) -> str:
    proc = _run_git(root, "diff", "--cached", "-U0", "--")
    return proc.stdout or ""


def git_staged_files(root: Path) -> list[str]:
    proc = _run_git(root, "diff", "--cached", "--name-only", "-z")
    if proc.returncode != 0 or not proc.stdout:
        return []
    return [p for p in proc.stdout.split("\0") if p]


def run_gate(
    root: Path,
    message: str,
    diff_text: str,
    staged_files: list[str],
    *,
    skip_artifacts: bool = False,
    history_rollback: bool = True,
    history_depth: int = 200,
) -> int:
    if not skip_artifacts:
        missing = missing_artifacts(root)
        if missing:
            print("VIOLATION (rule 53 logic-rollback-requires-approval): artifacts missing")
            for item in missing:
                print(f"  - {item}")
            return 1

    added = parse_unified_added(diff_text)
    hits = scan_added_lines(added)
    hits.extend(scan_revert_message(message, staged_files))
    if history_rollback:
        try:
            hits.extend(scan_history_rollback(root, added, history_depth=history_depth))
        except (subprocess.SubprocessError, OSError):
            # No git history / read-only env: fall back to keyword markers only.
            pass
    if not hits:
        print("OK: no logic-rollback markers (constraint 53).")
        return 0
    if has_waiver(message):
        print("OK: Logic-Rollback-OK waiver present for rollback markers.")
        return 0

    print("VIOLATION (rule 53 / ADR-0021): logic rollback without approval")
    for path, reason in hits[:20]:
        print(f"  - {path}: {reason}")
    print("  Fix: delete the old path (default), or get human approval and add")
    print("       `Logic-Rollback-OK: <reason>; remove-by: YYYY-MM-DD` in code")
    print("       AND `Logic-Rollback-OK: <reason>` in the commit message.")
    return 1


def read_commit_message(root: Path, msgfile: str | None) -> str:
    if msgfile:
        try:
            return Path(msgfile).read_text(encoding="utf-8", errors="replace")
        except OSError:
            return ""
    edit = root / ".git" / "COMMIT_EDITMSG"
    if edit.is_file():
        return edit.read_text(encoding="utf-8", errors="replace")
    return ""


def main(argv: list[str] | None = None) -> int:
    parser = argparse.ArgumentParser(
        description="Gate: logic rollback requires Logic-Rollback-OK approval.",
    )
    parser.add_argument("msgfile", nargs="?", default=None, help="commit-msg hook $1")
    parser.add_argument("--root", default=None)
    parser.add_argument("--commit-msg-file", default=None)
    parser.add_argument("--message", default=None)
    parser.add_argument("--diff", default=None, help="unified diff text (overrides git)")
    parser.add_argument("--staged-files", default=None, help="newline-separated paths")
    parser.add_argument("--skip-artifacts", action="store_true")
    parser.add_argument(
        "--no-history-rollback",
        action="store_true",
        help="disable git-history silent-rollback detection (OPT-20260819-043)",
    )
    parser.add_argument("--history-depth", type=int, default=200)
    args = parser.parse_args(argv)

    root = Path(args.root).resolve() if args.root else Path(__file__).resolve().parents[3]
    message = args.message
    if message is None:
        message = read_commit_message(root, args.msgfile or args.commit_msg_file)

    if args.diff is not None:
        diff_text = args.diff
    else:
        diff_text = git_cached_diff(root)

    if args.staged_files is not None:
        staged = [p.strip() for p in args.staged_files.splitlines() if p.strip()]
    else:
        staged = git_staged_files(root)

    return run_gate(
        root,
        message,
        diff_text,
        staged,
        skip_artifacts=args.skip_artifacts,
        history_rollback=not args.no_history_rollback,
        history_depth=args.history_depth,
    )


if __name__ == "__main__":
    raise SystemExit(main())
