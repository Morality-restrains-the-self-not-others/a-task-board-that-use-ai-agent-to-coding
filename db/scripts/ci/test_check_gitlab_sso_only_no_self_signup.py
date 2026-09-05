#!/usr/bin/env python3
"""Self-test for check_gitlab_sso_only_no_self_signup (no pytest required)."""

from __future__ import annotations

import importlib.util
import sys
import tempfile
from pathlib import Path

ROOT = Path(__file__).resolve().parents[3]
CHECKER = ROOT / "db" / "scripts" / "ci" / "check_gitlab_sso_only_no_self_signup.py"


def _load():
    spec = importlib.util.spec_from_file_location(
        "check_gitlab_sso_only_no_self_signup", CHECKER
    )
    assert spec and spec.loader
    mod = importlib.util.module_from_spec(spec)
    sys.modules[spec.name] = mod
    spec.loader.exec_module(mod)
    return mod


GOOD_CONF = (
    "signupEnabled: false\n"
    "passwordAuthWeb: false\n"
    "passwordAuthGit: false\n"
)

GOOD_COMPOSE = """
        gitlab_rails['signup_enabled'] = ${GITLAB_SIGNUP_ENABLED:-false}
        gitlab_rails['password_authentication_enabled_for_web'] = ${GITLAB_PASSWORD_AUTH_WEB:-false}
        gitlab_rails['password_authentication_enabled_for_git'] = ${GITLAB_PASSWORD_AUTH_GIT:-false}
        gitlab_rails['omniauth_enabled'] = true
        gitlab_rails['omniauth_allow_single_sign_on'] = ['openid_connect']
        gitlab_rails['omniauth_block_auto_created_users'] = false
"""

GOOD_LOADER = (
    "DEFAULT_SIGNUP_ENABLED = False\n"
    "DEFAULT_PASSWORD_AUTH_WEB = False\n"
    "DEFAULT_PASSWORD_AUTH_GIT = False\n"
)

GOOD_APPLY = "s.update!(\n  signup_enabled: $SIGNUP,\n)\n"
GOOD_RUN = '"$SCRIPT_DIR/scripts/apply_auth_policy.sh"\n'


def _tree(tmp: Path) -> None:
    (tmp / "conf" / "infra" / "git-service").mkdir(parents=True)
    (tmp / "conf" / "infra" / "git-service" / "config.yaml").write_text(
        GOOD_CONF, encoding="utf-8"
    )
    (tmp / "gitService" / "scripts").mkdir(parents=True)
    (tmp / "gitService" / "docker-compose.yml").write_text(GOOD_COMPOSE, encoding="utf-8")
    (tmp / "gitService" / "scripts" / "load_gitservice_config.py").write_text(
        GOOD_LOADER, encoding="utf-8"
    )
    (tmp / "gitService" / "scripts" / "apply_auth_policy.sh").write_text(
        GOOD_APPLY, encoding="utf-8"
    )
    (tmp / "gitService" / "run.sh").write_text(GOOD_RUN, encoding="utf-8")


def test_explicit_false_accepts_bool_and_string() -> None:
    mod = _load()
    assert mod.is_explicit_false(False)
    assert mod.is_explicit_false("false")
    assert mod.is_explicit_false("False")
    assert not mod.is_explicit_false(True)
    assert not mod.is_explicit_false("true")
    assert not mod.is_explicit_false(None)


def test_conf_all_false_ok() -> None:
    mod = _load()
    hits = mod.check_git_service_config(
        Path("conf/infra/git-service/config.yaml"),
        {
            "signupEnabled": False,
            "passwordAuthWeb": False,
            "passwordAuthGit": False,
        },
    )
    assert hits == []


def test_conf_signup_true_is_violation() -> None:
    mod = _load()
    hits = mod.check_git_service_config(
        Path("conf/infra/git-service/config.yaml"),
        {
            "signupEnabled": True,
            "passwordAuthWeb": False,
            "passwordAuthGit": False,
        },
    )
    assert any("signupEnabled" in h for h in hits)


def test_conf_missing_key_is_violation() -> None:
    mod = _load()
    hits = mod.check_git_service_config(
        Path("conf/infra/git-service-x/config.yaml"),
        {"signupEnabled": False, "passwordAuthWeb": False},
    )
    assert any("passwordAuthGit" in h and "missing" in h for h in hits)


def test_compose_ok() -> None:
    mod = _load()
    assert mod.check_compose_text("gitService/docker-compose.yml", GOOD_COMPOSE) == []


def test_compose_true_fallback_is_violation() -> None:
    mod = _load()
    text = GOOD_COMPOSE.replace(
        "${GITLAB_SIGNUP_ENABLED:-false}", "${GITLAB_SIGNUP_ENABLED:-true}"
    )
    hits = mod.check_compose_text("gitService/docker-compose.yml", text)
    assert any("GITLAB_SIGNUP_ENABLED" in h for h in hits)


def test_compose_omniauth_disabled_is_violation() -> None:
    mod = _load()
    text = GOOD_COMPOSE.replace("omniauth_enabled'] = true", "omniauth_enabled'] = false")
    hits = mod.check_compose_text("gitService/docker-compose.yml", text)
    assert any("omniauth_enabled" in h for h in hits)


def test_fixture_tree_clean() -> None:
    mod = _load()
    with tempfile.TemporaryDirectory() as td:
        tmp = Path(td)
        _tree(tmp)
        hits = mod.collect_violations(tmp, check_meta=False)
    assert hits == [], hits


def test_run_sh_helper_invokes_apply_auth_policy() -> None:
    """run.sh 可 source 容器脚本调用 apply_auth_policy.sh（行数削减后）。"""
    mod = _load()
    with tempfile.TemporaryDirectory() as td:
        tmp = Path(td)
        _tree(tmp)
        (tmp / "gitService" / "run.sh").write_text(
            'source "$SCRIPT_DIR/scripts/ensure_gitlab_container.sh"\n',
            encoding="utf-8",
        )
        (tmp / "gitService" / "scripts" / "ensure_gitlab_container.sh").write_text(
            GOOD_RUN, encoding="utf-8"
        )
        hits = mod.collect_violations(tmp, check_meta=False)
    assert hits == [], hits


def test_regional_compose_true_fallback_is_violation() -> None:
    mod = _load()
    with tempfile.TemporaryDirectory() as td:
        tmp = Path(td)
        _tree(tmp)
        bad = GOOD_COMPOSE.replace(
            "${GITLAB_SIGNUP_ENABLED:-false}", "${GITLAB_SIGNUP_ENABLED:-true}"
        )
        (tmp / "gitService" / "docker-compose.tencent-sh-1.yml").write_text(
            bad, encoding="utf-8"
        )
        hits = mod.collect_violations(tmp, check_meta=False)
    assert any(
        "docker-compose.tencent-sh-1.yml" in h and "GITLAB_SIGNUP_ENABLED" in h
        for h in hits
    ), hits


def test_fixture_tree_region_signup_true() -> None:
    mod = _load()
    with tempfile.TemporaryDirectory() as td:
        tmp = Path(td)
        _tree(tmp)
        region = tmp / "conf" / "infra" / "git-service-tencent-sh-1"
        region.mkdir(parents=True)
        (region / "config.yaml").write_text(
            "signupEnabled: true\npasswordAuthWeb: false\npasswordAuthGit: false\n",
            encoding="utf-8",
        )
        hits = mod.collect_violations(tmp, check_meta=False)
    assert any("signupEnabled" in h for h in hits)


def test_live_repo_passes() -> None:
    mod = _load()
    hits = mod.collect_violations(ROOT, check_meta=True)
    assert hits == [], hits


def test_meta_files_exist() -> None:
    mod = _load()
    for rel in mod.META_FILES:
        path = ROOT / rel
        assert path.is_file(), f"missing meta file: {rel}"


if __name__ == "__main__":
    tests = [v for k, v in sorted(globals().items()) if k.startswith("test_")]
    for t in tests:
        t()
        print(f"PASS {t.__name__}")
    print(f"ok ({len(tests)} tests)")
