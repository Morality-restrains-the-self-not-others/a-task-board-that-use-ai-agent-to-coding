#!/usr/bin/env python3
"""Unit tests for delete_merged_feat_branches helpers (no network)."""

from __future__ import annotations

import importlib.util
import os
import subprocess
import sys
import tempfile
import unittest
from pathlib import Path

# git 运行 hooks 时注入 GIT_DIR 等环境变量（gitlink 子仓时为
# .git/modules/<name>），子进程继承后 `git init` 会重新初始化宿主仓库的
# gitdir（core.bare 污染，见 db 同类修复）。本测试在临时目录创建仓库，
# 必须剥离全部 GIT_* 变量保证隔离。
CLEAN_ENV = {k: v for k, v in os.environ.items() if not k.startswith("GIT_")}


def load_mod():
    path = Path(__file__).resolve().parents[1] / "delete_merged_feat_branches.py"
    spec = importlib.util.spec_from_file_location("delete_merged_feat_branches", path)
    assert spec and spec.loader
    mod = importlib.util.module_from_spec(spec)
    sys.modules[spec.name] = mod
    spec.loader.exec_module(mod)
    return mod


class DeleteMergedFeatBranchesTest(unittest.TestCase):
    def test_monorepo_root_points_above_runAll(self):
        mod = load_mod()
        root = mod.monorepo_root()
        self.assertTrue((root / "runAll").is_dir())
        self.assertTrue((root / "runAll" / "scripts" / "delete_merged_feat_branches.py").is_file())

    def test_list_git_repos_includes_root_when_git_present(self):
        mod = load_mod()
        with tempfile.TemporaryDirectory() as td:
            root = Path(td)
            (root / ".git").mkdir()
            nested = root / "demo"
            nested.mkdir()
            (nested / ".git").mkdir()
            wt = root / "demo-wt"
            wt.mkdir()
            (wt / ".git").write_text("gitdir: /tmp/fake/worktrees/demo-wt\n")
            found = mod.list_git_repos(root)
            self.assertEqual(len(found), 2)
            self.assertEqual(found[0].resolve(), root.resolve())
            self.assertNotIn("demo-wt", {p.name for p in found})

    def test_branch_has_unique_diff_detects_empty(self):
        mod = load_mod()
        with tempfile.TemporaryDirectory() as td:
            repo = Path(td)
            for args in (
                ["git", "init"],
                ["git", "checkout", "-b", "main"],
                ["git", "commit", "--allow-empty", "-m", "init"],
                ["git", "branch", "archive/stash-test"],
            ):
                subprocess.run(args, cwd=repo, env=CLEAN_ENV, check=True, capture_output=True)
            self.assertFalse(mod.branch_has_unique_diff(repo, "archive/stash-test"))


if __name__ == "__main__":
    unittest.main()
