#!/usr/bin/env python3
"""Unit tests for cleanup_stale_worktrees helpers (no network)."""

from __future__ import annotations

import importlib.util
import os
import sys
import tempfile
import unittest
from pathlib import Path

CLEAN_ENV = {k: v for k, v in os.environ.items() if not k.startswith("GIT_")}


def load_mod(name: str, filename: str):
    path = Path(__file__).resolve().parents[1] / filename
    spec = importlib.util.spec_from_file_location(name, path)
    assert spec and spec.loader
    mod = importlib.util.module_from_spec(spec)
    sys.modules[name] = mod
    spec.loader.exec_module(mod)
    return mod


class CleanupStaleWorktreesTest(unittest.TestCase):
    def test_parse_worktrees_skips_primary_via_extra(self):
        mod = load_mod("cleanup_stale_worktrees", "cleanup_stale_worktrees.py")
        porcelain = """worktree /tmp/ram-work
HEAD abc
branch refs/heads/main

worktree /tmp/ram-work/conf-wt
HEAD def
branch refs/heads/feat/terminal-release-comment-csc
"""
        entries = mod.parse_worktrees(porcelain)
        self.assertEqual(len(entries), 2)
        self.assertEqual(entries[1]["branch"], "feat/terminal-release-comment-csc")
        self.assertTrue(mod.matches_shipped(entries[1], "feat/terminal-release-comment-csc"))
        self.assertFalse(mod.matches_shipped(entries[1], "feat/other"))

    def test_list_git_repos_skips_wt_suffix(self):
        feat = load_mod("delete_merged_feat_branches", "delete_merged_feat_branches.py")
        with tempfile.TemporaryDirectory() as td:
            root = Path(td)
            (root / ".git").mkdir()
            nested = root / "demo"
            nested.mkdir()
            (nested / ".git").mkdir()
            wt = root / "demo-wt"
            wt.mkdir()
            (wt / ".git").write_text("gitdir: /tmp/fake/worktrees/demo-wt\n")
            found = feat.list_git_repos(root)
            names = {p.name for p in found}
            self.assertIn("demo", names)
            self.assertNotIn("demo-wt", names)

    def test_apply_requires_shipped_branch(self):
        mod = load_mod("cleanup_stale_worktrees_main", "cleanup_stale_worktrees.py")
        argv = sys.argv
        sys.argv = ["cleanup_stale_worktrees.py", "--apply"]
        try:
            self.assertEqual(mod.main(), 2)
        finally:
            sys.argv = argv


class DeleteMergedFeatSkipWtTest(unittest.TestCase):
    def test_linked_gitdir_file_is_skipped(self):
        feat = load_mod("delete_merged_feat_branches_skip", "delete_merged_feat_branches.py")
        with tempfile.TemporaryDirectory() as td:
            root = Path(td)
            (root / ".git").mkdir()
            extra = root / "ram-work-meta2"
            extra.mkdir()
            (extra / ".git").write_text("gitdir: /tmp/fake/worktrees/meta2\n")
            found = feat.list_git_repos(root)
            self.assertEqual([p.resolve() for p in found], [root.resolve()])

    def test_submodule_gitdir_is_kept(self):
        feat = load_mod("delete_merged_feat_branches_sub", "delete_merged_feat_branches.py")
        with tempfile.TemporaryDirectory() as td:
            root = Path(td)
            (root / ".git").mkdir()
            nested = root / "conf"
            nested.mkdir()
            (nested / ".git").write_text("gitdir: ../.git/modules/conf\n")
            found = feat.list_git_repos(root)
            self.assertEqual({p.name for p in found if p != root}, {"conf"})


if __name__ == "__main__":
    unittest.main()
