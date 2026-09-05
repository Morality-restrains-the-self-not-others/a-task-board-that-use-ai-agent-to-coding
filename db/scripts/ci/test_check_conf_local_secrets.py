#!/usr/bin/env python3
"""Self-test for check_conf_local_secrets."""
from __future__ import annotations

import importlib.util
import sys
from pathlib import Path

ROOT = Path(__file__).resolve().parents[3]
CHECKER = ROOT / "db" / "scripts" / "ci" / "check_conf_local_secrets.py"


def _load():
    spec = importlib.util.spec_from_file_location("check_conf_local_secrets", CHECKER)
    assert spec and spec.loader
    mod = importlib.util.module_from_spec(spec)
    sys.modules[spec.name] = mod
    spec.loader.exec_module(mod)
    return mod


def test_live_repo_passes() -> None:
    hits = _load().collect_violations(ROOT)
    assert hits == [], hits


def test_detects_secret_key() -> None:
    mod = _load()
    hits = mod._secret_hits({"clientSecret": "not-empty-secret", "clientId": "public"})
    assert hits == ["clientSecret"], hits


def test_detects_internal_secret_and_skips_auth_flags() -> None:
    mod = _load()
    assert mod._is_secret_key("internalSecret")
    assert mod._is_secret_key("host_password")
    assert mod._is_secret_key("TASK2APP_SSO_JWT_SECRET")
    assert not mod._is_secret_key("passwordAuthWeb")
    assert not mod._is_secret_key("GITLAB_PASSWORD_AUTH_WEB")
    assert not mod._is_secret_key("accessTokenTTL")
    hits = mod._secret_hits(
        {
            "internalSecret": "x",
            "GITLAB_OIDC_CLIENT_SECRET": "${GITLAB_OIDC_CLIENT_SECRET:-dev}",
        }
    )
    assert hits == ["internalSecret"], hits


def test_lists_conf_when_parent_git_dir_is_set() -> None:
    import os

    mod = _load()
    old = os.environ.get("GIT_DIR")
    os.environ["GIT_DIR"] = "/tmp/not-a-git-dir"
    try:
        hits = mod.collect_violations(ROOT)
        assert not any("cannot list conf git files" in h for h in hits), hits
    finally:
        if old is None:
            os.environ.pop("GIT_DIR", None)
        else:
            os.environ["GIT_DIR"] = old


def test_registry_password_is_secret_hit() -> None:
    mod = _load()
    hits = mod._secret_hits({"mysql": {"host": "127.0.0.1", "password": "taskapp123"}})
    assert hits == ["mysql.password"], hits


def test_registry_empty_password_is_not_hit() -> None:
    mod = _load()
    hits = mod._secret_hits({"mysql": {"host": "127.0.0.1", "password": ""}})
    assert hits == [], hits


def test_pwd_md_is_forbidden_secret_doc() -> None:
    mod = _load()
    assert mod._is_forbidden_secret_doc("infra/git-service/gitLabRootPwd.md")
    assert mod._is_forbidden_secret_doc("RootPassword.md")
    assert not mod._is_forbidden_secret_doc("infra/git-service/config.yaml")


def main() -> int:
    test_live_repo_passes()
    test_detects_secret_key()
    test_detects_internal_secret_and_skips_auth_flags()
    test_lists_conf_when_parent_git_dir_is_set()
    test_registry_password_is_secret_hit()
    test_registry_empty_password_is_not_hit()
    test_pwd_md_is_forbidden_secret_doc()
    print("test_check_conf_local_secrets ok")
    return 0


if __name__ == "__main__":
    sys.exit(main())
