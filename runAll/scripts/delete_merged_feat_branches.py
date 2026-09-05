#!/usr/bin/env python3
"""删除已合入 main 的 feat/* 本地与远端分支（monorepo 多仓，含根仓）。

细则：.ai/01_project_constraints/21_merged_feat_branch_cleanup.md

用法:
  python3 runAll/scripts/delete_merged_feat_branches.py --dry-run
  python3 runAll/scripts/delete_merged_feat_branches.py --apply
  python3 runAll/scripts/delete_merged_feat_branches.py --apply --repo task2app
  python3 runAll/scripts/delete_merged_feat_branches.py --apply --archive-prefix archive/stash
"""

from __future__ import annotations

import argparse
import os
import subprocess
import sys
from pathlib import Path


PROTECTED_DEFAULT = ("main", "master")
PREFIX_DEFAULT = "feat/"
ARCHIVE_PREFIX_DEFAULT = "archive/stash"


def run(cwd: Path, *args: str) -> subprocess.CompletedProcess[str]:
    # git 运行 hooks 或从外部进程继承时会携带 GIT_DIR 等环境变量（gitlink
    # 子仓时为 .git/modules/<name>）。本脚本始终针对 cwd 仓库执行 git，剥离
    # 全部 GIT_* 变量，防止 git 命令被外部 GIT_DIR 重定向（core.bare 污染根因，
    # 见 db scripts/ci/test_check_bug_fix_unit_tests.py 同类修复）。
    clean_env = {k: v for k, v in os.environ.items() if not k.startswith("GIT_")}
    return subprocess.run(
        args,
        cwd=cwd,
        env=clean_env,
        capture_output=True,
        text=True,
        check=False,
    )


def monorepo_root() -> Path:
    here = Path(__file__).resolve()
    # runAll/scripts/this.py → parents[2] = monorepo root
    return here.parents[2]


def is_linked_worktree_checkout(path: Path) -> bool:
    """True for `{name}-wt/` or a checkout whose gitdir points at `.../worktrees/...`.

    Submodule `.git` files (`gitdir: ../.git/modules/<name>`) must still count as repos.
    """
    if path.name.endswith("-wt"):
        return True
    git = path / ".git"
    if not git.is_file():
        return False
    try:
        text = git.read_text(encoding="utf-8", errors="replace").lstrip()
    except OSError:
        return False
    if not text.lower().startswith("gitdir:"):
        return False
    gitdir = text.split(":", 1)[1].strip().replace("\\", "/")
    return "/worktrees/" in gitdir


def list_git_repos(root: Path) -> list[Path]:
    """Return git repos under monorepo: root repo first, then nested dirs with .git.

    Linked worktrees (`*-wt/`, extra meta checkouts) are not independent repos —
    they share the parent's git dir and must not be scanned as submodule-like roots.
    """
    repos: list[Path] = []
    root = root.resolve()
    if (root / ".git").exists():
        repos.append(root)
    for p in sorted(root.iterdir()):
        if not p.is_dir():
            continue
        if p.resolve() == root:
            continue
        if is_linked_worktree_checkout(p):
            continue
        if (p / ".git").exists():
            repos.append(p)
    return repos


def repo_label(repo: Path, root: Path) -> str:
    if repo.resolve() == root.resolve():
        return "(root)"
    return repo.name


def has_main(repo: Path) -> bool:
    return run(repo, "git", "rev-parse", "--verify", "main").returncode == 0


def current_branch(repo: Path) -> str:
    r = run(repo, "git", "branch", "--show-current")
    return (r.stdout or "").strip()


def list_branches(repo: Path, prefix: str) -> list[str]:
    r = run(repo, "git", "branch", "--format=%(refname:short)")
    out = []
    for line in (r.stdout or "").splitlines():
        b = line.strip()
        if b.startswith(prefix):
            out.append(b)
    return out


def local_feat_branches(repo: Path, prefix: str) -> list[str]:
    return list_branches(repo, prefix)


def remote_branches(repo: Path, remote: str, prefix: str) -> list[str]:
    r = run(repo, "git", "branch", "-r", "--format=%(refname:short)")
    out = []
    head = f"{remote}/"
    for line in (r.stdout or "").splitlines():
        ref = line.strip()
        if not ref.startswith(head):
            continue
        name = ref[len(head) :]
        if name == "HEAD" or "->" in name:
            continue
        if name.startswith(prefix):
            out.append(name)
    return out


def commits_not_in_main(repo: Path, branch: str) -> int:
    r = run(repo, "git", "rev-list", "--count", f"main..{branch}")
    if r.returncode != 0:
        return -1
    try:
        return int((r.stdout or "0").strip() or "0")
    except ValueError:
        return -1


def branch_has_unique_diff(repo: Path, branch: str) -> bool:
    """True when branch tip differs from main (non-empty diff), excluding merge commits only."""
    r = run(repo, "git", "diff", "--quiet", f"main...{branch}")
    if r.returncode == 0:
        return False
    if r.returncode == 1:
        return True
    # fallback: unknown diff state — treat as unique to be safe
    return True


def remotes(repo: Path) -> list[str]:
    r = run(repo, "git", "remote")
    return [x.strip() for x in (r.stdout or "").splitlines() if x.strip()]


def ensure_on_main(repo: Path, apply: bool) -> bool:
    cur = current_branch(repo)
    if cur == "main":
        return True
    if not apply:
        print(f"  [dry-run] would checkout main (now on {cur or 'DETACHED'})")
        return True
    r = run(repo, "git", "checkout", "main")
    if r.returncode != 0:
        print(f"  SKIP checkout main: {(r.stderr or r.stdout).strip()}")
        return False
    return True


def delete_local(repo: Path, branch: str, apply: bool) -> None:
    if not apply:
        print(f"  [dry-run] git branch -d {branch}")
        return
    wt = run(repo, "git", "worktree", "list", "--porcelain")
    if wt.returncode == 0 and f"branch refs/heads/{branch}" in (wt.stdout or ""):
        print(f"  keep local {branch} (in use by worktree)")
        return
    r = run(repo, "git", "branch", "-d", branch)
    if r.returncode != 0:
        r2 = run(repo, "git", "branch", "-D", branch)
        if r2.returncode != 0:
            err = (r2.stderr or r.stderr or "").strip()
            if "used by worktree" in err:
                print(f"  keep local {branch} (in use by worktree)")
            else:
                print(f"  FAIL local delete {branch}: {err}")
        else:
            print(f"  deleted local {branch} (-D)")
    else:
        print(f"  deleted local {branch}")


def delete_remote(repo: Path, remote: str, branch: str, apply: bool) -> None:
    if not apply:
        print(f"  [dry-run] git push {remote} --delete {branch}")
        return
    r = run(repo, "git", "push", remote, "--delete", branch)
    if r.returncode != 0:
        err = (r.stderr or r.stdout or "").strip()
        if "remote ref does not exist" in err.lower() or "not found" in err.lower():
            print(f"  remote {remote}/{branch} already gone")
        else:
            print(f"  FAIL remote delete {remote}/{branch}: {err}")
    else:
        print(f"  deleted remote {remote}/{branch}")


def should_delete_branch(repo: Path, branch: str, *, archive_prefix: str) -> tuple[bool, str]:
    if branch.startswith(archive_prefix):
        if branch_has_unique_diff(repo, branch):
            return False, "archive branch still has unique diff vs main"
        return True, "archive/stash tip has no unique diff — delete without merge"
    n = commits_not_in_main(repo, branch)
    if n != 0:
        return False, f"main..branch={n}"
    return True, "fully merged into main"


def process_repo(
    repo: Path,
    root: Path,
    *,
    prefix: str,
    archive_prefix: str,
    protect: set[str],
    apply: bool,
    fetch: bool,
    include_archive: bool,
) -> int:
    if not has_main(repo):
        print(f"SKIP {repo_label(repo, root)}: no main")
        return 0
    print(f"=== {repo_label(repo, root)} ===")
    if not ensure_on_main(repo, apply):
        return 0

    deleted = 0
    prefixes = [prefix]
    if include_archive and archive_prefix:
        prefixes.append(archive_prefix)

    for branch_prefix in prefixes:
        for branch in list_branches(repo, branch_prefix):
            if branch in protect:
                continue
            ok, reason = should_delete_branch(repo, branch, archive_prefix=archive_prefix)
            if not ok:
                print(f"  keep local {branch} ({reason})")
                continue
            if branch.startswith(archive_prefix):
                print(f"  archive {branch}: {reason}")
            delete_local(repo, branch, apply)
            deleted += 1

    for remote in remotes(repo):
        if fetch:
            run(repo, "git", "fetch", remote, "--prune")
        main_ref = f"{remote}/main"
        if run(repo, "git", "rev-parse", "--verify", main_ref).returncode != 0:
            continue
        for branch_prefix in prefixes:
            for branch in remote_branches(repo, remote, branch_prefix):
                if branch in protect:
                    continue
                ok, reason = should_delete_branch(repo, f"{remote}/{branch}", archive_prefix=archive_prefix)
                if not ok:
                    print(f"  keep {remote}/{branch} ({reason})")
                    continue
                if branch.startswith(archive_prefix):
                    print(f"  archive {remote}/{branch}: {reason}")
                delete_remote(repo, remote, branch, apply)
                deleted += 1

    return deleted


def main() -> int:
    parser = argparse.ArgumentParser(description="Delete merged feat/* branches across monorepo")
    parser.add_argument("--apply", action="store_true", help="Actually delete (default is dry-run)")
    parser.add_argument("--dry-run", action="store_true", help="Preview only (default)")
    parser.add_argument("--fetch", action="store_true", help="git fetch --prune each remote before scanning")
    parser.add_argument("--repo", action="append", default=[], help="Limit to repo dir name or (root)")
    parser.add_argument("--prefix", default=PREFIX_DEFAULT, help="Branch name prefix (default feat/)")
    parser.add_argument(
        "--archive-prefix",
        default=ARCHIVE_PREFIX_DEFAULT,
        help="Archive/stash branch prefix to delete when tip has no unique diff (default archive/stash)",
    )
    parser.add_argument(
        "--include-archive",
        action="store_true",
        help="Also scan/delete archive/stash-* branches (no empty merge into main)",
    )
    parser.add_argument(
        "--protect",
        default=",".join(PROTECTED_DEFAULT),
        help="Comma-separated branch names never deleted",
    )
    parser.add_argument(
        "--root",
        default="",
        help="Monorepo root (default: inferred from script path)",
    )
    args = parser.parse_args()
    apply = bool(args.apply) and not bool(args.dry_run)

    root = Path(args.root).resolve() if args.root else monorepo_root()
    protect = {x.strip() for x in args.protect.split(",") if x.strip()}
    repos = list_git_repos(root)
    if args.repo:
        wanted = set(args.repo)
        filtered: list[Path] = []
        for r in repos:
            label = repo_label(r, root)
            if r.name in wanted or label in wanted or "(root)" in wanted and r.resolve() == root.resolve():
                filtered.append(r)
        repos = filtered

    if not repos:
        print("No git repos found", file=sys.stderr)
        return 1

    mode = "APPLY" if apply else "DRY-RUN"
    print(
        f"[{mode}] root={root} prefix={args.prefix!r} archive={args.archive_prefix!r} "
        f"include_archive={args.include_archive} fetch={args.fetch} repos={len(repos)}"
    )
    total = 0
    for repo in repos:
        total += process_repo(
            repo,
            root,
            prefix=args.prefix,
            archive_prefix=args.archive_prefix,
            protect=protect,
            apply=apply,
            fetch=bool(args.fetch),
            include_archive=bool(args.include_archive),
        )
    print(f"[{mode}] done, actions={total}")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
