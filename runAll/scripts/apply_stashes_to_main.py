#!/usr/bin/env python3
"""Apply git stash entries onto main with conflict resolution favoring main.

- Skips junk stashes (WIP on detached HEAD, merge/conflict noise).
- On conflict: checkout --ours (main) for conflicted paths, continue.
- pre-commit failure: abort apply and do NOT drop stash.

Usage:
  python3 runAll/scripts/apply_stashes_to_main.py --dry-run
  python3 runAll/scripts/apply_stashes_to_main.py --apply
  python3 runAll/scripts/apply_stashes_to_main.py --apply --repo task2app
"""

from __future__ import annotations

import argparse
import re
import subprocess
import sys
from pathlib import Path

JUNK_STASH_RE = re.compile(
    r"(?i)(^|\s)(wip on|index on|merge branch|checkout:|pull\.|rebase)",
)


def run(cwd: Path, *args: str) -> subprocess.CompletedProcess[str]:
    return subprocess.run(args, cwd=cwd, capture_output=True, text=True, check=False)


def monorepo_root() -> Path:
    return Path(__file__).resolve().parents[2]


def list_git_repos(root: Path) -> list[Path]:
    repos: list[Path] = []
    root = root.resolve()
    if (root / ".git").exists():
        repos.append(root)
    for p in sorted(root.iterdir()):
        if p.is_dir() and (p / ".git").exists() and p.resolve() != root:
            repos.append(p)
    return repos


def stash_list(repo: Path) -> list[tuple[int, str]]:
    r = run(repo, "git", "stash", "list", "--format=%gd|%s")
    if r.returncode != 0:
        return []
    out: list[tuple[int, str]] = []
    for line in (r.stdout or "").splitlines():
        if "|" not in line:
            continue
        ref, msg = line.split("|", 1)
        m = re.search(r"stash@\{(\d+)\}", ref)
        if not m:
            continue
        out.append((int(m.group(1)), msg.strip()))
    return out


def is_junk_stash(message: str) -> bool:
    return bool(JUNK_STASH_RE.search(message))


def ensure_main(repo: Path, apply: bool) -> bool:
    cur = run(repo, "git", "branch", "--show-current")
    branch = (cur.stdout or "").strip()
    if branch == "main":
        return True
    if not apply:
        print(f"  [dry-run] would checkout main (now on {branch or 'DETACHED'})")
        return True
    r = run(repo, "git", "checkout", "main")
    if r.returncode != 0:
        print(f"  SKIP: checkout main failed: {(r.stderr or r.stdout).strip()}")
        return False
    return True


def resolve_conflicts_keep_main(repo: Path) -> bool:
    st = run(repo, "git", "status", "--porcelain")
    if st.returncode != 0:
        return False
    conflicted = []
    for line in (st.stdout or "").splitlines():
        if line.startswith("UU ") or line.startswith("AA ") or line.startswith("DU ") or line.startswith("UD "):
            conflicted.append(line[3:].strip())
    if not conflicted:
        return True
    for path in conflicted:
        run(repo, "git", "checkout", "--ours", "--", path)
        run(repo, "git", "add", "--", path)
    return True


def run_pre_commit(repo: Path) -> bool:
    hook = repo / ".git" / "hooks" / "pre-commit"
    if not hook.is_file():
        return True
    r = run(repo, str(hook))
    if r.returncode != 0:
        err = (r.stderr or r.stdout or "").strip()
        print(f"  pre-commit failed: {err}")
        return False
    return True


def apply_one_stash(repo: Path, index: int, message: str, apply: bool) -> str:
    if is_junk_stash(message):
        return "skipped-junk"
    if not apply:
        print(f"  [dry-run] would apply stash@{{{index}}}: {message}")
        return "dry-run"
    r = run(repo, "git", "stash", "apply", f"stash@{{{index}}}")
    if r.returncode != 0:
        err = (r.stderr or r.stdout or "").strip()
        if "No stash entries" in err:
            return "missing"
        print(f"  apply stash@{{{index}}} conflict/noise: {err}")
        resolve_conflicts_keep_main(repo)
        if run(repo, "git", "diff", "--cached", "--quiet").returncode == 0 and run(
            repo, "git", "diff", "--quiet"
        ).returncode == 0:
            run(repo, "git", "reset", "--hard", "HEAD")
            return "empty-after-ours"
    if not run_pre_commit(repo):
        run(repo, "git", "reset", "--hard", "HEAD")
        return "pre-commit-blocked"
    drop = run(repo, "git", "stash", "drop", f"stash@{{{index}}}")
    if drop.returncode != 0:
        print(f"  WARN: applied but drop failed: {(drop.stderr or drop.stdout).strip()}")
    return "applied"


def process_repo(repo: Path, root: Path, apply: bool) -> int:
    label = "(root)" if repo.resolve() == root.resolve() else repo.name
    print(f"=== {label} ===")
    if not ensure_main(repo, apply):
        return 0
    stashes = stash_list(repo)
    if not stashes:
        print("  no stashes")
        return 0
    actions = 0
    for index, message in stashes:
        result = apply_one_stash(repo, index, message, apply)
        print(f"  stash@{{{index}}}: {result} — {message}")
        if result in ("applied", "dry-run"):
            actions += 1
    return actions


def main() -> int:
    parser = argparse.ArgumentParser(description="Apply stashes onto main (conflicts keep main)")
    parser.add_argument("--apply", action="store_true")
    parser.add_argument("--dry-run", action="store_true")
    parser.add_argument("--repo", action="append", default=[])
    parser.add_argument("--root", default="")
    args = parser.parse_args()
    apply = bool(args.apply) and not args.dry_run
    root = Path(args.root).resolve() if args.root else monorepo_root()
    repos = list_git_repos(root)
    if args.repo:
        wanted = set(args.repo)
        repos = [
            r
            for r in repos
            if r.name in wanted or ("(root)" in wanted and r.resolve() == root.resolve())
        ]
    mode = "APPLY" if apply else "DRY-RUN"
    print(f"[{mode}] root={root} repos={len(repos)}")
    total = 0
    for repo in repos:
        total += process_repo(repo, root, apply)
    print(f"[{mode}] done, actions={total}")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
