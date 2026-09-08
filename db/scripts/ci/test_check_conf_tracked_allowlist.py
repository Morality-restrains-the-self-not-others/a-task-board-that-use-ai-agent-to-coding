#!/usr/bin/env python3
"""Self-test for check_conf_tracked_allowlist (no pytest required)."""

from __future__ import annotations

import importlib.util
import os
import sys
from pathlib import Path

ROOT = Path(__file__).resolve().parents[3]
CHECKER = ROOT / "db" / "scripts" / "ci" / "check_conf_tracked_allowlist.py"


def _load():
    spec = importlib.util.spec_from_file_location("check_conf_tracked_allowlist", CHECKER)
    assert spec and spec.loader
    mod = importlib.util.module_from_spec(spec)
    sys.modules[spec.name] = mod
    spec.loader.exec_module(mod)
    return mod


def test_allows_yaml_companion_sync_and_identity() -> None:
    mod = _load()
    for rel in (
        "base.yaml",
        "runAll.yaml",
        "auth/task-auth/sync.manifest.yaml",
        "auth/task-auth/sync.sh",
        "ai.md",
        "runAll.yaml.ai.md",
        "README.md",
        "LICENSE",
        ".gitignore",
        "core/sms/config.local.yaml.example",
        ".githooks/pre-commit",
    ):
        assert mod.is_allowed(rel), rel


def test_forbids_non_config_assets() -> None:
    mod = _load()
    for rel in (
        "com.user.ramsync.plist",
        "infra/git-service/test_config_resource_keys.py",
        "auth/git-oauth/test_provider_website_isolation.py",
        "infra/git-service-tencent-sh-1/docker-compose.yml",
        "taskChromePlugin/generateKey.sh",
        ".githooks/commit-msg",
        ".githooks/lib/random_test_runner.sh",
        ".githooks/install.sh",
        ".githooks/HOOK_VERSION",
        ".claude/settings.json",
        "scripts/ci/check_conf_sync.sh",
    ):
        assert not mod.is_allowed(rel), rel


def test_dot_gitignore_is_not_stripped() -> None:
    """lstrip('./') would turn .gitignore into gitignore — must use prefix strip."""
    mod = _load()
    assert mod.is_allowed(".gitignore")
    assert not mod.is_allowed("gitignore")


def test_collect_violations_from_explicit_list() -> None:
    mod = _load()
    hits = mod.collect_violations_from(
        ["base.yaml", "com.user.ramsync.plist", ".githooks/commit-msg"]
    )
    assert hits == ["com.user.ramsync.plist", ".githooks/commit-msg"], hits


def test_live_conf_repo_passes() -> None:
    mod = _load()
    hits = mod.collect_violations_from(mod.list_conf_tracked(ROOT))
    assert hits == [], hits


def test_list_conf_tracked_ignores_caller_git_index_file() -> None:
    """pre-commit in db/other repos sets GIT_INDEX_FILE; must still list conf.git."""
    mod = _load()
    os.environ["GIT_INDEX_FILE"] = "/nonexistent/index"
    os.environ["GIT_DIR"] = "/nonexistent/git"
    try:
        rels = mod.list_conf_tracked(ROOT)
    finally:
        os.environ.pop("GIT_INDEX_FILE", None)
        os.environ.pop("GIT_DIR", None)
    assert "base.yaml" in rels
    assert ".githooks/HOOK_VERSION" not in rels


if __name__ == "__main__":
    test_allows_yaml_companion_sync_and_identity()
    test_forbids_non_config_assets()
    test_dot_gitignore_is_not_stripped()
    test_collect_violations_from_explicit_list()
    test_live_conf_repo_passes()
    test_list_conf_tracked_ignores_caller_git_index_file()
    print("ok")
