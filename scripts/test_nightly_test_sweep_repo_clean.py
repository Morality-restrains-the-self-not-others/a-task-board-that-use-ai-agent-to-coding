#!/usr/bin/env python3
"""Unit tests for nightly_test_sweep.repo_clean ignorable untracked noise.

LICENSE / README* 未跟踪不应阻塞夜间抽测；真实 WIP 仍须 SKIP。
"""
from __future__ import annotations

import json
import subprocess
import tempfile
import unittest
from pathlib import Path

import nightly_test_sweep as sweep


def _git(repo: Path, *args: str) -> None:
    subprocess.check_call(
        ["git", "-C", str(repo), *args],
        stdout=subprocess.DEVNULL,
        stderr=subprocess.DEVNULL,
    )


def _init_repo() -> Path:
    d = Path(tempfile.mkdtemp(prefix="sweep-clean-"))
    _git(d, "init")
    _git(d, "config", "user.email", "test@example.com")
    _git(d, "config", "user.name", "test")
    (d / "keep.txt").write_text("ok\n", encoding="utf-8")
    _git(d, "add", "keep.txt")
    _git(d, "commit", "-m", "init")
    return d


class RepoCleanTests(unittest.TestCase):
    def test_clean_worktree(self) -> None:
        repo = _init_repo()
        ok, reason = sweep.repo_clean(repo)
        self.assertTrue(ok)
        self.assertEqual(reason, "")

    def test_license_only_untracked_is_clean(self) -> None:
        repo = _init_repo()
        (repo / "LICENSE").write_text("MIT\n", encoding="utf-8")
        (repo / "README.md").write_text("x\n", encoding="utf-8")
        (repo / "README copy.md").write_text("x\n", encoding="utf-8")
        ok, reason = sweep.repo_clean(repo)
        self.assertTrue(ok, reason)
        self.assertEqual(reason, "")

    def test_real_modified_is_dirty(self) -> None:
        repo = _init_repo()
        (repo / "keep.txt").write_text("changed\n", encoding="utf-8")
        ok, reason = sweep.repo_clean(repo)
        self.assertFalse(ok)
        self.assertIn("dirty worktree", reason)

    def test_license_plus_real_wip_is_dirty(self) -> None:
        repo = _init_repo()
        (repo / "LICENSE").write_text("MIT\n", encoding="utf-8")
        (repo / "src.go").write_text("package main\n", encoding="utf-8")
        ok, reason = sweep.repo_clean(repo)
        self.assertFalse(ok)
        self.assertIn("dirty worktree", reason)
        self.assertIn("src.go", reason)

    def test_nested_license_untracked_is_clean(self) -> None:
        repo = _init_repo()
        nested = repo / "vendor" / "foo"
        nested.mkdir(parents=True)
        (nested / "LICENSE").write_text("MIT\n", encoding="utf-8")
        ok, reason = sweep.repo_clean(repo)
        self.assertTrue(ok, reason)


class SessionHubInteractiveTests(unittest.TestCase):
    """OPT-20260823-020: session_hub_has_interactive 只在有 kind=interactive 会话时为真。"""

    def setUp(self) -> None:
        # 临时 ROOT + 假 claude-agent 二进制，避免触碰真实 Session Hub
        self.tmp = Path(tempfile.mkdtemp(prefix="sweep-hub-"))
        (self.tmp / "claude-agent" / "bin").mkdir(parents=True)
        (self.tmp / "claude-agent" / "bin" / "claude-agent").touch()
        self._orig_root = sweep.ROOT
        self._orig_run = sweep.run
        sweep.ROOT = self.tmp

    def tearDown(self) -> None:
        sweep.ROOT = self._orig_root
        sweep.run = self._orig_run

    def _fake_run(self, rc: int = 0, stdout: str = "") -> None:
        def _run(cmd, cwd=None, timeout=None, input_text=None):
            return subprocess.CompletedProcess(cmd, rc, stdout=stdout, stderr="")
        sweep.run = _run

    def test_interactive_session_detected(self) -> None:
        self._fake_run(0, json.dumps([{"kind": "interactive", "session_id": "s1"}]))
        self.assertTrue(sweep.session_hub_has_interactive())

    def test_only_sweep_and_headless_not_interactive(self) -> None:
        self._fake_run(0, json.dumps([{"kind": "sweep"}, {"kind": "headless"}]))
        self.assertFalse(sweep.session_hub_has_interactive())

    def test_nonzero_returncode_false(self) -> None:
        self._fake_run(1, "")
        self.assertFalse(sweep.session_hub_has_interactive())

    def test_invalid_json_false(self) -> None:
        self._fake_run(0, "not-json")
        self.assertFalse(sweep.session_hub_has_interactive())

    def test_missing_binary_false(self) -> None:
        sweep.ROOT = Path(tempfile.mkdtemp(prefix="sweep-nobina-"))
        self.assertFalse(sweep.session_hub_has_interactive())


if __name__ == "__main__":
    unittest.main()
