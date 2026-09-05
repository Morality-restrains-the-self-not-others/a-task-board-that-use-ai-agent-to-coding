#!/usr/bin/env python3
"""CI：.gitmodules 列出的子仓若存在 .git，则须有 .githooks/ 入库真源（git-hooks-version-control v13）：
- .githooks/pre-commit（随机单测门禁，OPT-20260719-033 / v65 nightly-test-sweep 重构）
- .githooks/commit-msg + check_bug_fix_commit_msg.sh（bug-fix 回归单测门禁，OPT-20260805-002）
- .githooks/lib/random_test_runner.sh（共享抽测库）
- core.hooksPath == .githooks（Git 原生激活；.git/hooks/ 不再放置业务钩子副本）

设计: docs/superpowers/specs/2026-08-06-git-hooks-version-control-design.md
"""

from __future__ import annotations

import os
import subprocess
import sys
from pathlib import Path


def git_clean_env() -> dict[str, str]:
    env = os.environ.copy()
    for key in (
        "GIT_DIR",
        "GIT_INDEX_FILE",
        "GIT_WORK_TREE",
        "GIT_OBJECT_DIRECTORY",
        "GIT_PREFIX",
        "GIT_COMMON_DIR",
    ):
        env.pop(key, None)
    return env


def monorepo_root() -> Path:
    here = Path(__file__).resolve()
    for parent in here.parents:
        if (parent / ".gitmodules").is_file() and (parent / "db" / "registry.yaml").is_file():
            return parent
    raise FileNotFoundError(".gitmodules + db/registry.yaml not found")


def submodule_paths(root: Path) -> list[str]:
    out = subprocess.check_output(
        ["git", "config", "-f", str(root / ".gitmodules"), "--get-regexp", r"^submodule\..*\.path$"],
        text=True,
        env=git_clean_env(),
    )
    paths: list[str] = []
    for line in out.splitlines():
        parts = line.split(None, 1)
        if len(parts) == 2:
            paths.append(parts[1].strip())
    return sorted(set(paths))


def has_git_dir(repo: Path) -> bool:
    git = repo / ".git"
    return git.is_dir() or git.is_file()


def hooks_path_ok(repo: Path) -> bool:
    out = subprocess.run(
        ["git", "-C", str(repo), "config", "core.hooksPath"],
        capture_output=True, text=True, check=False,
        env=git_clean_env(),
    )
    return out.returncode == 0 and out.stdout.strip() == ".githooks"


REQUIRED_HOOKS = ("pre-commit", "commit-msg", "check_bug_fix_commit_msg.sh")
# 配置/依赖仓豁免模板钩子家族：conf=手写 oauth live-check pre-commit；sdk=第三方 vendored SDK
SKIP_HOOK_REPOS = frozenset({"conf", "sdk"})
# 仅 conf 要求手写 oauth live-check pre-commit；其余 SKIP 仓允许 .githooks 为空或手写
SKIP_REQUIRE_OAUTH_PRECOMMIT = frozenset({"conf"})
SKIP_FORBIDDEN_HOOK_FILES = (
    "commit-msg",
    "check_bug_fix_commit_msg.sh",
    "install.sh",
    "HOOK_VERSION",
)


def check_skip_repo(rel: str, repo: Path) -> list[str]:
    """Config/vendored repos: no template hook family (OPT-20260901-022).

    conf keeps a hand-written oauth live-check pre-commit; other SKIP repos
    (e.g. sdk) allow an empty .githooks or hand-written hooks only.
    """
    hits: list[str] = []
    for name in SKIP_FORBIDDEN_HOOK_FILES:
        if (repo / ".githooks" / name).is_file():
            hits.append(f"{rel}: unexpected .githooks/{name} (SKIP_HOOK_REPOS)")
    runner = repo / ".githooks" / "lib" / "random_test_runner.sh"
    if runner.is_file():
        hits.append(f"{rel}: unexpected .githooks/lib/random_test_runner.sh")
    if rel in SKIP_REQUIRE_OAUTH_PRECOMMIT:
        pre = repo / ".githooks" / "pre-commit"
        if not pre.is_file():
            hits.append(f"{rel}: missing .githooks/pre-commit (config quality hook)")
            return hits
        body = pre.read_text(encoding="utf-8")
        if "check_git_oauth_client_id_live.py" not in body:
            hits.append(f"{rel}: .githooks/pre-commit must call oauth live check")
    return hits


def main() -> int:
    root = monorepo_root()
    missing: list[str] = []
    skipped: list[str] = []
    ok = 0
    for rel in submodule_paths(root):
        repo = root / rel
        if not repo.is_dir():
            skipped.append(f"{rel}: directory missing")
            continue
        if not has_git_dir(repo):
            skipped.append(f"{rel}: no .git (not checked out)")
            continue
        if rel in SKIP_HOOK_REPOS:
            skip_hits = check_skip_repo(rel, repo)
            if skip_hits:
                missing.extend(skip_hits)
                continue
            if not hooks_path_ok(repo):
                missing.append(f"{rel}: core.hooksPath != .githooks (git config core.hooksPath .githooks)")
                continue
            ok += 1
            continue
        repo_missing = [h for h in REQUIRED_HOOKS if not (repo / ".githooks" / h).is_file()]
        if repo_missing:
            for h in repo_missing:
                missing.append(f"{rel}: missing .githooks/{h} (deploy: scripts/deploy_repo_random_precommit.sh)")
            continue
        if not (repo / ".githooks" / "lib" / "random_test_runner.sh").is_file():
            missing.append(f"{rel}: missing .githooks/lib/random_test_runner.sh (redeploy: deploy_repo_random_precommit.sh)")
            continue
        if not hooks_path_ok(repo):
            missing.append(f"{rel}: core.hooksPath != .githooks (activate: bash .githooks/install.sh)")
            continue
        ok += 1
    if missing:
        print("FAIL check_subrepo_random_precommit_hooks:", file=sys.stderr)
        for line in missing:
            print(f"  - {line}", file=sys.stderr)
        return 1
    print(f"OK check_subrepo_random_precommit_hooks (ok={ok} skipped={len(skipped)})")
    for line in skipped:
        print(f"  skip {line}")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
