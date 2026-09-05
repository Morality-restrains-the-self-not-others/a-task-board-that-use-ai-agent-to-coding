"""Unit tests for check_git_oauth_client_id_live.py (OPT-20260807-041a)."""

from __future__ import annotations

import importlib.util
import tempfile
from pathlib import Path

HERE = Path(__file__).resolve().parent
MOD_PATH = HERE / "check_git_oauth_client_id_live.py"


def _load():
    spec = importlib.util.spec_from_file_location("check_git_oauth_client_id_live", MOD_PATH)
    mod = importlib.util.module_from_spec(spec)
    assert spec.loader is not None
    spec.loader.exec_module(mod)
    return mod


# ---------- path / diff 筛选 ----------

def test_is_provider_path_accepts_git_oauth_tree() -> None:
    mod = _load()
    assert mod.is_provider_path("conf/auth/git-oauth/providers/http-github-com--app-daydaymoney.yaml")
    assert mod.is_provider_path("conf/auth/task-credential/git-oauth-providers/http-github-com.yaml")


def test_is_provider_path_rejects_ai_md_and_others() -> None:
    mod = _load()
    assert not mod.is_provider_path("conf/auth/git-oauth/providers/github.ai.md")
    assert not mod.is_provider_path("conf/auth/task-credential/git-oauth-providers/http-github-com.yaml.ai.md")
    assert not mod.is_provider_path("conf/auth/git-oauth/other.yaml")
    assert not mod.is_provider_path("taskFE/package.json")
    assert not mod.is_provider_path("conf/auth/git-oauth/providers/README")


def test_filter_provider_files_dedups_and_sorts() -> None:
    mod = _load()
    changed = [
        "taskFE/app/src/main.js",
        "conf/auth/git-oauth/providers/http-github-com--app-daydaymoney.yaml",
        "conf/auth/task-credential/git-oauth-providers/http-github-com.yaml",
        "conf/auth/git-oauth/providers/http-github-com--app-daydaymoney.yaml",
        "conf/auth/git-oauth/providers/github.ai.md",
    ]
    assert mod.filter_provider_files(changed) == [
        "conf/auth/git-oauth/providers/http-github-com--app-daydaymoney.yaml",
        "conf/auth/task-credential/git-oauth-providers/http-github-com.yaml",
    ]


# ---------- slug 推导 ----------

def test_derive_slug_from_service_provider() -> None:
    mod = _load()
    data = {"provider": "github", "service_provider": "github-official-daydaymoney"}
    assert mod.derive_slug(data, "http-github-com--app-daydaymoney.yaml") == "daydaymoney"


def test_derive_slug_from_filename_fallback() -> None:
    mod = _load()
    data = {"provider": "github"}
    assert mod.derive_slug(data, "http-github-com--app-daydaymoney.yaml") == "daydaymoney"


def test_derive_slug_none_for_gitlab() -> None:
    mod = _load()
    assert mod.derive_slug({"service_provider": "gitlab-daydaymoney"}, "http-gitlab-daydaymoney-com.yaml") is None


# ---------- API 响应判定 ----------

def test_validate_not_found_is_failure() -> None:
    mod = _load()
    status, detail = mod.validate_client_id((404, None, "HTTP 404"), "Iv23li4xi6ZBcq1LKZk6", "nope-app")
    assert status == "not_found"
    assert "does not exist" in detail


def test_validate_match_ok() -> None:
    mod = _load()
    status, _ = mod.validate_client_id(
        (200, {"client_id": "Iv23li4xi6ZBcq1LKZk6"}, None), "Iv23li4xi6ZBcq1LKZk6", "daydaymoney"
    )
    assert status == "ok"


def test_validate_mismatch_is_failure() -> None:
    mod = _load()
    status, detail = mod.validate_client_id(
        (200, {"client_id": "Iv23liOTHER"}, None), "Iv23li4xi6ZBcq1LKZk6", "daydaymoney"
    )
    assert status == "mismatch"
    assert "Iv23liOTHER" in detail


def test_validate_network_error_skips() -> None:
    mod = _load()
    status, detail = mod.validate_client_id((None, None, "URLError: <urlopen error timed out>"), "x", "daydaymoney")
    assert status == "skip"
    assert "network" in detail


def test_validate_rate_limit_skips() -> None:
    mod = _load()
    status, _ = mod.validate_client_id((403, None, "HTTP 403"), "x", "daydaymoney")
    assert status == "skip"


# ---------- 端到端（mock fetch_json，临时 conf 树） ----------

def test_validate_provider_path_ok_with_mocked_api() -> None:
    mod = _load()
    with tempfile.TemporaryDirectory() as td:
        root = Path(td)
        rel = "conf/auth/git-oauth/providers/http-github-com--app-daydaymoney.yaml"
        path = root / rel
        path.parent.mkdir(parents=True)
        path.write_text(
            "provider: github\n"
            "service_provider: github-official-daydaymoney\n"
            "target:\n"
            "  website: https://github.com\n"
            "  client_id: Iv23li4xi6ZBcq1LKZk6\n"
            "  client_secret: secret\n",
            encoding="utf-8",
        )
        orig = mod.fetch_json
        mod.fetch_json = lambda url, timeout=5.0: (200, {"client_id": "Iv23li4xi6ZBcq1LKZk6"}, None)
        try:
            status, detail = mod.validate_provider_path(root, rel)
        finally:
            mod.fetch_json = orig
        assert status == "ok"
        assert "client_id matches" in detail


def test_validate_provider_path_skips_gitlab() -> None:
    mod = _load()
    with tempfile.TemporaryDirectory() as td:
        root = Path(td)
        rel = "conf/auth/git-oauth/providers/http-gitlab-daydaymoney-com.yaml"
        path = root / rel
        path.parent.mkdir(parents=True)
        path.write_text(
            "provider: gitlab\nservice_provider: gitlab-daydaymoney\n"
            "target:\n  website: https://gitlab.daydaymoney.com\n",
            encoding="utf-8",
        )
        status, detail = mod.validate_provider_path(root, rel)
        assert status == "skip"
        assert "not a github.com provider" in detail


def test_main_no_diff_returns_zero() -> None:
    mod = _load()
    # --changed 空列表 -> 无目标 -> 直接 0，不触网
    assert mod.main(["--changed"]) == 0


if __name__ == "__main__":
    test_is_provider_path_accepts_git_oauth_tree()
    test_is_provider_path_rejects_ai_md_and_others()
    test_filter_provider_files_dedups_and_sorts()
    test_derive_slug_from_service_provider()
    test_derive_slug_from_filename_fallback()
    test_derive_slug_none_for_gitlab()
    test_validate_not_found_is_failure()
    test_validate_match_ok()
    test_validate_mismatch_is_failure()
    test_validate_network_error_skips()
    test_validate_rate_limit_skips()
    test_validate_provider_path_ok_with_mocked_api()
    test_validate_provider_path_skips_gitlab()
    test_main_no_diff_returns_zero()
    print("OK test_check_git_oauth_client_id_live")
