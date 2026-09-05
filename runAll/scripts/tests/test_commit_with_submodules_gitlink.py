"""OPT-20260819-020 回归：根仓 commit_repo 必须透传 include_untracked，
保证暂存的 gitlink（子仓指针）进入最终 commit，不被 pre-commit 索引改写吞掉。

测试用真实 submodule：父仓推进子仓 HEAD 后，gitlink 变更 + 跟踪文件 + 未跟踪文件
一起经 commit_repo 提交，断言最终 commit 全部包含。
"""
from __future__ import annotations

import os
import subprocess
from pathlib import Path

import commit_with_submodules as cws
import pytest


# git 运行 hooks 时注入 GIT_DIR 等；子进程继承后 `git init`/`git add` 会打到宿主
# gitdir（core.bare 污染，OPT-20260806-059）。临时仓测试必须剥离 GIT_*。
def _clean_git_env() -> dict[str, str]:
    return {k: v for k, v in os.environ.items() if not k.startswith("GIT_")}


@pytest.fixture(autouse=True)
def _isolate_git_env(monkeypatch):
    for key in [k for k in os.environ if k.startswith("GIT_")]:
        monkeypatch.delenv(key, raising=False)


def _git(path: Path, *args: str, check: bool = True) -> subprocess.CompletedProcess[str]:
    return subprocess.run(
        ["git", *args],
        cwd=path,
        capture_output=True,
        text=True,
        check=check,
        env=_clean_git_env(),
    )


def _git_ok(path: Path, *args: str) -> str:
    r = _git(path, *args)
    assert r.returncode == 0, f"git {' '.join(args)} failed: {r.stderr}"
    return r.stdout.strip()


@pytest.fixture
def gitlink_repo(tmp_path: Path):
    """父仓 + 子仓，子仓作为 submodule 登记。"""
    sub = tmp_path / "subrepo"
    sub.mkdir()
    _git_ok(sub, "init", "-q")
    _git_ok(sub, "config", "user.email", "t@t")
    _git_ok(sub, "config", "user.name", "t")
    (sub / "a.txt").write_text("v1", encoding="utf-8")
    _git_ok(sub, "add", "a.txt")
    _git_ok(sub, "commit", "-q", "-m", "feat: sub v1")

    parent = tmp_path / "parent"
    parent.mkdir()
    _git_ok(parent, "init", "-q")
    _git_ok(parent, "config", "user.email", "t@t")
    _git_ok(parent, "config", "user.name", "t")
    (parent / "tracked.txt").write_text("t1", encoding="utf-8")
    _git_ok(parent, "add", "tracked.txt")
    _git_ok(parent, "commit", "-q", "-m", "feat: parent base")
    # 登记 submodule（file protocol 需显式 allow）
    _git_ok(parent, "-c", "protocol.file.allow=always", "submodule", "add", "-q", str(sub), "sub")
    _git_ok(parent, "commit", "-q", "-m", "chore: add submodule")
    return parent, sub


def _advance_submodule_and_stage(parent: Path, sub: Path) -> None:
    """推进子仓 HEAD 并在父仓暂存 gitlink + 改跟踪文件 + 未跟踪文件。"""
    (sub / "a.txt").write_text("v2", encoding="utf-8")
    _git_ok(sub, "commit", "-q", "-am", "feat: sub v2")
    # 父仓登记新子仓指针（git add sub 会记录 gitlink mode 160000）
    _git_ok(parent, "add", "sub")
    (parent / "tracked.txt").write_text("t2", encoding="utf-8")
    _git_ok(parent, "add", "tracked.txt")
    (parent / "untracked.txt").write_text("u1", encoding="utf-8")


def test_root_commit_repo_include_untracked_keeps_gitlinks(gitlink_repo):
    parent, sub = gitlink_repo
    _advance_submodule_and_stage(parent, sub)

    before = _git_ok(parent, "rev-parse", "HEAD")
    ok = cws.commit_repo(
        "(root)", parent, "chore: sync submodule pointers",
        apply=True, push=False, include_untracked=True,
    )
    assert ok, "commit_repo with include_untracked must succeed"

    after = _git_ok(parent, "rev-parse", "HEAD")
    assert after != before, "expected a new root commit"

    # 最终 commit 必须同时含 gitlink 变更、跟踪文件变更、未跟踪文件
    files = _git_ok(parent, "show", "--name-only", "--format=", "HEAD").split()
    assert "tracked.txt" in files, "tracked change missing from final commit"
    assert "untracked.txt" in files, "untracked file missing from final commit"

    tree = _git_ok(parent, "ls-tree", "HEAD", "sub")
    assert tree.startswith("160000"), f"sub must be a gitlink (160000), got: {tree}"


def test_root_commit_repo_without_include_untracked_skips_untracked(gitlink_repo):
    """不带 include_untracked 时未跟踪文件不进 commit，但 gitlink 仍进（回归对比）。"""
    parent, sub = gitlink_repo
    _advance_submodule_and_stage(parent, sub)

    ok = cws.commit_repo(
        "(root)", parent, "chore: sync submodule pointers",
        apply=True, push=False, include_untracked=False,
    )
    assert ok
    files = _git_ok(parent, "show", "--name-only", "--format=", "HEAD").split()
    assert "untracked.txt" not in files, "untracked must not be committed without include_untracked"
    assert "tracked.txt" in files
    tree = _git_ok(parent, "ls-tree", "HEAD", "sub")
    assert tree.startswith("160000"), f"gitlink must be committed, got: {tree}"
