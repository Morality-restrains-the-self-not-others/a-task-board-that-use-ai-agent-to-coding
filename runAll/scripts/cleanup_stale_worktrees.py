#!/usr/bin/env python3
"""Remove leftover git worktrees after a feature ships (or scan them).

`*-wt/` checkouts and extra meta worktrees (e.g. ram-work-meta2) must not
outlive the feature. A unique SHA on the feat branch is not a keep signal
once `--shipped-branch` is passed — content may have landed on main via
another commit.

细则：.ai/01_project_constraints/21_merged_feat_branch_cleanup.md

用法:
  python3 runAll/scripts/cleanup_stale_worktrees.py --scan
  python3 runAll/scripts/cleanup_stale_worktrees.py --apply --shipped-branch feat/foo
"""

from __future__ import annotations

import argparse
import os
import subprocess
import sys
from pathlib import Path

SCAN_REL = Path(".runall") / "stale_worktrees.txt"
_SCRIPTS = Path(__file__).resolve().parent
if str(_SCRIPTS) not in sys.path:
    sys.path.insert(0, str(_SCRIPTS))

from delete_merged_feat_branches import list_git_repos  # noqa: E402


def run(cwd: Path, *args: str) -> subprocess.CompletedProcess[str]:
    clean_env = {k: v for k, v in os.environ.items() if not k.startswith("GIT_")}
    if not cwd.exists():
        return subprocess.CompletedProcess(args, 1, "", f"cwd missing: {cwd}")
    return subprocess.run(
        args,
        cwd=cwd,
        env=clean_env,
        capture_output=True,
        text=True,
        check=False,
    )


def monorepo_root() -> Path:
    return Path(__file__).resolve().parents[2]


def parse_worktrees(porcelain: str) -> list[dict[str, str]]:
    entries: list[dict[str, str]] = []
    cur: dict[str, str] = {}
    for line in porcelain.splitlines():
        if line.startswith("worktree "):
            if cur:
                entries.append(cur)
            cur = {"path": line[len("worktree ") :], "branch": "", "head": ""}
        elif line.startswith("HEAD "):
            cur["head"] = line[len("HEAD ") :]
        elif line.startswith("branch "):
            ref = line[len("branch ") :]
            prefix = "refs/heads/"
            cur["branch"] = ref[len(prefix) :] if ref.startswith(prefix) else ref
        elif line == "detached":
            cur["detached"] = "1"
        elif line == "bare":
            cur["bare"] = "1"
        elif line == "locked" or line.startswith("locked "):
            cur["locked"] = "1"
    if cur:
        entries.append(cur)
    return entries


def extra_worktrees(repo: Path) -> list[dict[str, str]]:
    r = run(repo, "git", "worktree", "list", "--porcelain")
    if r.returncode != 0:
        return []
    top = run(repo, "git", "rev-parse", "--show-toplevel")
    primary = Path((top.stdout or str(repo)).strip()).resolve()
    common_raw = (run(repo, "git", "rev-parse", "--git-common-dir").stdout or "").strip()
    common = Path(common_raw)
    common = common.resolve() if common.is_absolute() else (repo / common).resolve()
    out: list[dict[str, str]] = []
    for wt in parse_worktrees(r.stdout or ""):
        if wt.get("bare") == "1":
            continue
        path = Path(wt["path"]).resolve()
        if path == primary or path == common:
            continue
        # Submodule main checkout is recorded as .git/modules/<name>; skip git internals.
        sp = str(path)
        if "/.git/" in sp or sp.endswith("/.git"):
            continue
        if not path.exists():
            continue
        wt["path"] = sp
        out.append(wt)
    return out


def is_dirty(path: Path) -> bool:
    if not path.exists():
        return False
    r = run(path, "git", "status", "--porcelain")
    if r.returncode != 0:
        return True
    return bool((r.stdout or "").strip())


def matches_shipped(wt: dict[str, str], shipped: str) -> bool:
    return bool(shipped) and wt.get("branch") == shipped


def orphan_wt_dirs(root: Path) -> list[Path]:
    return [p.resolve() for p in sorted(root.iterdir()) if p.is_dir() and p.name.endswith("-wt")]


def write_scan(root: Path, rows: list[str]) -> Path:
    out = root / SCAN_REL
    out.parent.mkdir(parents=True, exist_ok=True)
    out.write_text(("\n".join(rows) + "\n") if rows else "", encoding="utf-8")
    return out


def repo_has_branch(repo: Path, branch: str) -> bool:
    if run(repo, "git", "show-ref", "--verify", "--quiet", f"refs/heads/{branch}").returncode == 0:
        return True
    rem = run(repo, "git", "remote")
    for remote in (x.strip() for x in (rem.stdout or "").splitlines() if x.strip()):
        if run(repo, "git", "rev-parse", "--verify", f"{remote}/{branch}").returncode == 0:
            return True
    return False


def collect_rows(root: Path) -> list[str]:
    rows: list[str] = []
    seen: set[str] = set()
    for repo in list_git_repos(root):
        run(repo, "git", "worktree", "prune")
        label = "(root)" if repo.resolve() == root.resolve() else repo.name
        for wt in extra_worktrees(repo):
            path = wt["path"]
            if path in seen:
                continue
            seen.add(path)
            dirty = "dirty" if is_dirty(Path(path)) else "clean"
            rows.append(f"{path}\t{label}\t{wt.get('branch') or '(detached)'}\t{dirty}")
    for p in orphan_wt_dirs(root):
        sp = str(p)
        if sp in seen:
            continue
        seen.add(sp)
        dirty = "dirty" if (p / ".git").exists() and is_dirty(p) else "orphan"
        rows.append(f"{sp}\t(orphan-wt)\t?\t{dirty}")
    return rows


def remove_one(repo: Path, path: Path, apply: bool) -> str:
    if not apply:
        return f"  [dry-run] git worktree remove {path}"
    r = run(repo, "git", "worktree", "remove", str(path))
    if r.returncode != 0:
        r = run(repo, "git", "worktree", "remove", "--force", str(path))
    if r.returncode != 0:
        return f"  FAIL remove {path}: {(r.stderr or r.stdout).strip()}"
    run(repo, "git", "worktree", "prune")
    return f"  removed {path}"


def delete_shipped_branch(repo: Path, branch: str, apply: bool) -> list[str]:
    notes: list[str] = []
    local = run(repo, "git", "show-ref", "--verify", "--quiet", f"refs/heads/{branch}")
    if local.returncode == 0:
        if apply:
            r = run(repo, "git", "branch", "-D", branch)
            if r.returncode == 0:
                notes.append(f"  deleted local {branch}")
            else:
                notes.append(f"  FAIL local {branch}: {(r.stderr or r.stdout).strip()}")
        else:
            notes.append(f"  [dry-run] git branch -D {branch}")
    rem = run(repo, "git", "remote")
    for remote in (x.strip() for x in (rem.stdout or "").splitlines() if x.strip()):
        if run(repo, "git", "rev-parse", "--verify", f"{remote}/{branch}").returncode != 0:
            continue
        if apply:
            r = run(repo, "git", "push", remote, "--delete", branch)
            if r.returncode == 0:
                notes.append(f"  deleted remote {remote}/{branch}")
            else:
                err = (r.stderr or r.stdout or "").strip()
                if (
                    "remote ref does not exist" in err.lower()
                    or "not found" in err.lower()
                    or "远程引用不存在" in err
                ):
                    notes.append(f"  remote {remote}/{branch} already gone")
                else:
                    notes.append(f"  FAIL remote {remote}/{branch}: {err}")
        else:
            notes.append(f"  [dry-run] git push {remote} --delete {branch}")
    return notes


def apply_shipped(root: Path, shipped: str, apply: bool) -> int:
    actions = 0
    for repo in list_git_repos(root):
        extras = extra_worktrees(repo)
        matching = [wt for wt in extras if matches_shipped(wt, shipped)]
        if not matching and not repo_has_branch(repo, shipped):
            continue
        label = "(root)" if repo.resolve() == root.resolve() else repo.name
        print(f"=== {label} ===")
        for wt in extras:
            path = Path(wt["path"])
            if not matches_shipped(wt, shipped):
                print(f"  keep {path} (branch {wt.get('branch') or '(detached)'} != {shipped})")
                continue
            if is_dirty(path):
                print(f"  keep {path} (dirty; commit or stash before remove)")
                continue
            print(remove_one(repo, path, apply))
            actions += 1
        for note in delete_shipped_branch(repo, shipped, apply):
            print(note)
            actions += 1
    return actions


def main() -> int:
    parser = argparse.ArgumentParser(description="Scan or remove leftover git worktrees")
    parser.add_argument("--scan", action="store_true", help="Write .runall/stale_worktrees.txt")
    parser.add_argument("--apply", action="store_true", help="Remove worktrees / delete shipped branch")
    parser.add_argument("--dry-run", action="store_true", help="Preview --apply without changing anything")
    parser.add_argument("--shipped-branch", default="", help="feat/* (or other) branch that has landed on main")
    parser.add_argument("--root", default="", help="Monorepo root")
    args = parser.parse_args()
    root = Path(args.root).resolve() if args.root else monorepo_root()

    if args.apply and not args.shipped_branch:
        print("--apply requires --shipped-branch <feat/name>", file=sys.stderr)
        return 2

    if args.apply:
        mode = "DRY-RUN" if args.dry_run else "APPLY"
        print(f"[{mode}] shipped-branch={args.shipped_branch!r} root={root}")
        n = apply_shipped(root, args.shipped_branch, apply=not args.dry_run)
        print(f"[{mode}] done, actions={n}")
        rows = collect_rows(root)
        write_scan(root, rows)
        return 0

    rows = collect_rows(root)
    out = write_scan(root, rows)
    print(f"[SCAN] {len(rows)} leftover worktree(s) → {out}")
    for row in rows:
        print(f"  {row}")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
