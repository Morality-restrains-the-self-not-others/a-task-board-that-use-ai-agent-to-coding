#!/usr/bin/env python3
"""Self-test for check_subrepo_random_precommit_hooks skip-repo branch."""

from __future__ import annotations

import importlib.util
import sys
import tempfile
from pathlib import Path

ROOT = Path(__file__).resolve().parents[3]
CHECKER = ROOT / "db" / "scripts" / "ci" / "check_subrepo_random_precommit_hooks.py"


def _load():
    spec = importlib.util.spec_from_file_location(
        "check_subrepo_random_precommit_hooks", CHECKER
    )
    assert spec and spec.loader
    mod = importlib.util.module_from_spec(spec)
    sys.modules[spec.name] = mod
    spec.loader.exec_module(mod)
    return mod


def test_conf_is_skip_hook_repo() -> None:
    mod = _load()
    assert "conf" in mod.SKIP_HOOK_REPOS


def test_thin_hook_ok() -> None:
    mod = _load()
    with tempfile.TemporaryDirectory() as tmp:
        repo = Path(tmp)
        hook = repo / ".githooks" / "pre-commit"
        hook.parent.mkdir(parents=True)
        hook.write_text(
            "#!/bin/bash\n"
            "LIVE_CHECK=\"$META_ROOT/db/scripts/ci/check_git_oauth_client_id_live.py\"\n"
            "python3 \"$LIVE_CHECK\" --all\n",
            encoding="utf-8",
        )
        assert mod.check_skip_repo("conf", repo) == []


def test_thin_hook_rejects_template_family() -> None:
    mod = _load()
    with tempfile.TemporaryDirectory() as tmp:
        repo = Path(tmp)
        githooks = repo / ".githooks"
        githooks.mkdir(parents=True)
        (githooks / "pre-commit").write_text("#!/bin/bash\necho skip\n", encoding="utf-8")
        (githooks / "commit-msg").write_text("#!/bin/bash\n", encoding="utf-8")
        (githooks / "lib").mkdir()
        (githooks / "lib" / "random_test_runner.sh").write_text("#\n", encoding="utf-8")
        hits = mod.check_skip_repo("conf", repo)
        assert any("commit-msg" in h for h in hits), hits
        assert any("random_test_runner" in h for h in hits), hits
        assert any("oauth live check" in h for h in hits), hits


def test_sdk_is_skip_hook_repo() -> None:
    mod = _load()
    assert "sdk" in mod.SKIP_HOOK_REPOS


def test_vendored_skip_repo_allows_empty_githooks() -> None:
    mod = _load()
    with tempfile.TemporaryDirectory() as tmp:
        repo = Path(tmp)
        # sdk 无手写钩子要求：空 .githooks 应通过
        assert mod.check_skip_repo("sdk", repo) == []


def test_vendored_skip_repo_rejects_template_family() -> None:
    mod = _load()
    with tempfile.TemporaryDirectory() as tmp:
        repo = Path(tmp)
        githooks = repo / ".githooks"
        githooks.mkdir(parents=True)
        (githooks / "commit-msg").write_text("#!/bin/bash\n", encoding="utf-8")
        (githooks / "lib").mkdir()
        (githooks / "lib" / "random_test_runner.sh").write_text("#\n", encoding="utf-8")
        hits = mod.check_skip_repo("sdk", repo)
        assert any("commit-msg" in h for h in hits), hits
        assert any("random_test_runner" in h for h in hits), hits
        # sdk 不需要 conf 的 oauth live-check pre-commit
        assert not any("oauth" in h for h in hits), hits


def test_deploy_script_skips_conf_and_sdk() -> None:
    text = (ROOT / "scripts" / "deploy_repo_random_precommit.sh").read_text(encoding="utf-8")
    assert 'SKIP_HOOK_REPOS="conf sdk"' in text
    assert "SKIP_HOOK" in text


if __name__ == "__main__":
    for name, fn in sorted(globals().items()):
        if name.startswith("test_") and callable(fn):
            fn()
            print(f"PASS {name}")
    print("ok")
