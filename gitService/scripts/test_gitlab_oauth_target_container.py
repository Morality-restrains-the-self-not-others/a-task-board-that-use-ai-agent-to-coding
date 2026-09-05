#!/usr/bin/env python3
"""Unit tests: GitLab OAuth app must sync to the CE instance that owns the website."""
from __future__ import annotations

import sys
from pathlib import Path

SCRIPT = Path(__file__).resolve().parent / "gitlab_oauth_target_container.py"
sys.path.insert(0, str(SCRIPT.parent))
import gitlab_oauth_target_container as g

REPO_ROOT = Path(__file__).resolve().parents[2]


def test_hostname_of_strips_path():
    assert g.hostname_of("https://gitlab-tencent-sh-1.daydaymoney.com/group/repo") == (
        "gitlab-tencent-sh-1.daydaymoney.com"
    )
    assert g.hostname_of("gitlab.daydaymoney.com") == "gitlab.daydaymoney.com"


def test_resolve_container_for_website_matches_host_only():
    instances = [
        ("gitlab.daydaymoney.com", "gitlab"),
        ("gitlab-tencent-sh-1.daydaymoney.com", "gitlab-tencent-sh-1"),
    ]
    assert g.resolve_container_for_website(
        "https://gitlab-tencent-sh-1.daydaymoney.com/ljy/ram-work", instances
    ) == "gitlab-tencent-sh-1"
    assert g.resolve_container_for_website("https://gitlab.daydaymoney.com/g/r", instances) == "gitlab"
    assert g.resolve_container_for_website("https://gitlab.example.org/g/r", instances) is None


def test_real_repo_tencent_sh1_maps_to_named_container():
    container = g.resolve_container(
        "${scheme}://${subdomains.gitlabTencentSh1}", REPO_ROOT
    )
    assert container == "gitlab-tencent-sh-1"


def test_real_repo_default_gitlab_maps_to_gitlab_container():
    container = g.resolve_container("${scheme}://${subdomains.gitlab}", REPO_ROOT)
    assert container == "gitlab"


def test_placeholder_regex_matches_camel_case_subdomain():
    raw = "${scheme}://${subdomains.gitlabTencentSh1}/oauth"
    keys = [m.group(1) for m in g._PLACEHOLDER_RE.finditer(raw)]
    assert "scheme" in keys
    assert "subdomains.gitlabTencentSh1" in keys


def test_overlay_conf_local_wins_public_url():
    import tempfile

    with tempfile.TemporaryDirectory() as tmp:
        root = Path(tmp)
        (root / "conf").mkdir(parents=True)
        (root / "conf" / "base.yaml").write_text(
            "scheme: https\nbaseDomain: example.test\nsubdomains: {}\n",
            encoding="utf-8",
        )
        inst = root / "conf" / "infra" / "git-service"
        inst.mkdir(parents=True)
        (inst / "config.yaml").write_text(
            "publicUrl: https://tracked.example.test\ncontainerName: gitlab\n",
            encoding="utf-8",
        )
        local = root / "conf-local" / "infra" / "git-service"
        local.mkdir(parents=True)
        (local / "config.yaml").write_text(
            "publicUrl: https://overlay.example.test\n",
            encoding="utf-8",
        )
        instances = g.git_service_instances(root)
        assert instances == [("overlay.example.test", "gitlab")], instances


if __name__ == "__main__":
    tests = [
        test_hostname_of_strips_path,
        test_resolve_container_for_website_matches_host_only,
        test_real_repo_tencent_sh1_maps_to_named_container,
        test_real_repo_default_gitlab_maps_to_gitlab_container,
        test_placeholder_regex_matches_camel_case_subdomain,
        test_overlay_conf_local_wins_public_url,
    ]
    failed = 0
    for fn in tests:
        try:
            fn()
            print(f"PASS {fn.__name__}")
        except AssertionError as exc:
            failed += 1
            print(f"FAIL {fn.__name__}: {exc}")
    sys.exit(1 if failed else 0)
