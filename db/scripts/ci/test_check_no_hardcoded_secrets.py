#!/usr/bin/env python3
"""Self-test for check_no_hardcoded_secrets (no pytest required)."""

from __future__ import annotations

import importlib.util
import sys
import tempfile
from pathlib import Path

ROOT = Path(__file__).resolve().parents[3]
CHECKER = ROOT / "db" / "scripts" / "ci" / "check_no_hardcoded_secrets.py"


def _load():
    spec = importlib.util.spec_from_file_location("check_no_hardcoded_secrets", CHECKER)
    assert spec and spec.loader
    mod = importlib.util.module_from_spec(spec)
    sys.modules[spec.name] = mod
    spec.loader.exec_module(mod)
    return mod


def _write(path: Path, text: str) -> None:
    path.parent.mkdir(parents=True, exist_ok=True)
    path.write_text(text, encoding="utf-8")


def _meta(tmp: Path) -> None:
    for rel in (
        ".ai/01_project_constraints/62_no_hardcoded_secrets.md",
        ".cursor/rules/no-hardcoded-secrets.mdc",
        "docs/adr/0046-no-hardcoded-secrets.md",
    ):
        _write(tmp / rel, "# meta\n")


def _stripe_live() -> str:
    return "sk_live_" + ("w" * 24)


def _aws_id() -> str:
    return "AKIA" + "ABCDEFGHIJKLMNOP"


def _gh_pat() -> str:
    return "ghp_" + ("A" * 36)


def _pem_block() -> str:
    return (
        "-----BEGIN PRIVATE KEY-----
REDACTED
-----END PRIVATE KEY-----\n"
    )


def test_hardcoded_password_in_go_fails() -> None:
    mod = _load()
    with tempfile.TemporaryDirectory() as raw:
        tmp = Path(raw)
        _write(
            tmp / "taskBill" / "src" / "pay.go",
            'package main\n\nconst dbPassword = "hunter2-prod-like-credential"\n',
        )
        hits = mod.collect_violations(tmp, check_meta=False)
        assert any("assignment" in h and "pay.go" in h for h in hits), hits


def test_getenv_assignment_passes() -> None:
    mod = _load()
    with tempfile.TemporaryDirectory() as raw:
        tmp = Path(raw)
        _write(
            tmp / "taskBill" / "src" / "pay.go",
            "package main\n\n"
            'import "os"\n\n'
            "func secret() string { return os.Getenv(\"DB_PASSWORD\") }\n",
        )
        hits = mod.collect_violations(tmp, check_meta=False)
        assert hits == [], hits


def test_conf_yaml_secret_passes() -> None:
    mod = _load()
    with tempfile.TemporaryDirectory() as raw:
        tmp = Path(raw)
        _write(
            tmp / "conf" / "auth" / "task-auth" / "config.yaml",
            "internalSecret: hunter2-prod-like-credential\n",
        )
        hits = mod.collect_violations(tmp, check_meta=False)
        assert hits == [], hits


def test_pem_in_source_fails() -> None:
    mod = _load()
    with tempfile.TemporaryDirectory() as raw:
        tmp = Path(raw)
        _write(tmp / "taskAuth" / "src" / "tls.go", "package main\n\nconst k = `" + _pem_block() + "`\n")
        hits = mod.collect_violations(tmp, check_meta=False)
        assert any("pem-private-key" in h for h in hits), hits


def test_pem_in_test_file_still_fails() -> None:
    mod = _load()
    with tempfile.TemporaryDirectory() as raw:
        tmp = Path(raw)
        _write(
            tmp / "taskAuth" / "src" / "tls_test.go",
            "package main\n\nconst fixture = `" + _pem_block() + "`\n",
        )
        hits = mod.collect_violations(tmp, check_meta=False)
        assert any("pem-private-key" in h for h in hits), hits


def test_placeholder_in_unit_test_passes() -> None:
    mod = _load()
    with tempfile.TemporaryDirectory() as raw:
        tmp = Path(raw)
        _write(
            tmp / "taskBill" / "src" / "pay_test.go",
            'package main\n\nconst apiKey = "test-api-key"\n',
        )
        hits = mod.collect_violations(tmp, check_meta=False)
        assert hits == [], hits


def test_aws_access_key_fails() -> None:
    mod = _load()
    with tempfile.TemporaryDirectory() as raw:
        tmp = Path(raw)
        _write(
            tmp / "taskCloudService" / "src" / "oss.go",
            f"package main\n\nconst id = \"{_aws_id()}\"\n",
        )
        hits = mod.collect_violations(tmp, check_meta=False)
        assert any("aws-access-key" in h for h in hits), hits


def test_well_known_aws_example_passes() -> None:
    mod = _load()
    with tempfile.TemporaryDirectory() as raw:
        tmp = Path(raw)
        _write(
            tmp / "taskCloudService" / "src" / "oss.go",
            'package main\n\nconst id = "AKIAIOSFODNN7EXAMPLE"\n',
        )
        hits = mod.collect_violations(tmp, check_meta=False)
        assert hits == [], hits


def test_stripe_live_fails() -> None:
    mod = _load()
    with tempfile.TemporaryDirectory() as raw:
        tmp = Path(raw)
        _write(
            tmp / "taskBill" / "src" / "stripe.go",
            f"package main\n\nconst k = \"{_stripe_live()}\"\n",
        )
        hits = mod.collect_violations(tmp, check_meta=False)
        assert any("stripe-live" in h for h in hits), hits


def test_github_pat_fails() -> None:
    mod = _load()
    with tempfile.TemporaryDirectory() as raw:
        tmp = Path(raw)
        _write(
            tmp / "taskGitOauth" / "src" / "gh.go",
            f"package main\n\nconst t = \"{_gh_pat()}\"\n",
        )
        hits = mod.collect_violations(tmp, check_meta=False)
        assert any("github-pat" in h for h in hits), hits


def test_waiver_comment_passes() -> None:
    mod = _load()
    with tempfile.TemporaryDirectory() as raw:
        tmp = Path(raw)
        _write(
            tmp / "taskBill" / "src" / "pay.go",
            "package main\n\n"
            "// Secret-Hardcode-OK: documented local fixture; not a live credential\n"
            'const dbPassword = "hunter2-prod-like-credential"\n',
        )
        hits = mod.collect_violations(tmp, check_meta=False)
        assert hits == [], hits


def test_third_party_skipped() -> None:
    mod = _load()
    with tempfile.TemporaryDirectory() as raw:
        tmp = Path(raw)
        _write(
            tmp / "taskAuth" / "third_party" / "kafka-go" / "dialer.go",
            f"package kafka\n\nconst k = `{_pem_block()}`\n",
        )
        hits = mod.collect_violations(tmp, check_meta=False)
        assert hits == [], hits


def test_gitlab_ce_skipped() -> None:
    mod = _load()
    with tempfile.TemporaryDirectory() as raw:
        tmp = Path(raw)
        _write(
            tmp / "gitService" / "gitlab-ce" / "spec" / "secret.rb",
            f"TOKEN = '{_gh_pat()}'\n",
        )
        hits = mod.collect_violations(tmp, check_meta=False)
        assert hits == [], hits


def test_placeholder_literal_in_prod_passes() -> None:
    mod = _load()
    with tempfile.TemporaryDirectory() as raw:
        tmp = Path(raw)
        _write(
            tmp / "taskBill" / "src" / "pay.go",
            'package main\n\nconst password = "changeme"\n',
        )
        hits = mod.collect_violations(tmp, check_meta=False)
        assert hits == [], hits


def test_missing_meta_fails() -> None:
    mod = _load()
    with tempfile.TemporaryDirectory() as raw:
        tmp = Path(raw)
        hits = mod.collect_violations(tmp, check_meta=True)
        assert any("missing meta file" in h for h in hits), hits


def test_meta_present_empty_tree_passes() -> None:
    mod = _load()
    with tempfile.TemporaryDirectory() as raw:
        tmp = Path(raw)
        _meta(tmp)
        hits = mod.collect_violations(tmp, check_meta=True)
        assert hits == [], hits


def test_dotenv_on_disk_not_scanned() -> None:
    """Local .env is an allowed secret store; only committed dotenv is a leak."""
    mod = _load()
    with tempfile.TemporaryDirectory() as raw:
        tmp = Path(raw)
        _write(tmp / ".env", f"STRIPE_KEY={_stripe_live()}\n")
        hits = mod.collect_violations(tmp, check_meta=False)
        assert hits == [], hits


def test_is_placeholder_helpers() -> None:
    mod = _load()
    assert mod.is_placeholder("test-api-key")
    assert mod.is_placeholder("changeme")
    assert mod.is_placeholder("${DB_PASSWORD}")
    assert mod.is_placeholder("$ACCESS_TOKEN")
    assert mod.is_placeholder("__TASK2APP_ACCESS_TOKEN__")
    assert not mod.is_placeholder("hunter2-prod-like-credential")


def test_vue_kebab_password_prop_passes() -> None:
    mod = _load()
    with tempfile.TemporaryDirectory() as raw:
        tmp = Path(raw)
        _write(
            tmp / "taskFE" / "app" / "src" / "views" / "Login.vue",
            "<template>\n"
            '  <LoginEmailPasswordFields :show-email-password="showEmailPassword" />\n'
            "</template>\n",
        )
        hits = mod.collect_violations(tmp, check_meta=False)
        assert hits == [], hits


def test_cjk_error_message_assignment_passes() -> None:
    mod = _load()
    with tempfile.TemporaryDirectory() as raw:
        tmp = Path(raw)
        _write(
            tmp / "taskFE" / "app" / "src" / "views" / "ResetPassword.vue",
            "errors.value.password = '密码强度不足请包含字母数字和特殊字符'\n",
        )
        hits = mod.collect_violations(tmp, check_meta=False)
        assert hits == [], hits


def test_example_config_placeholder_passes() -> None:
    """*.example config keeps assignment placeholder exemption."""
    mod = _load()
    with tempfile.TemporaryDirectory() as raw:
        tmp = Path(raw)
        _write(
            tmp / "trae-agent" / "onlineServiceJS" / "trae_config.yaml.example",
            "api_key: your_openai_api_key\nclient_secret: your-client-secret-value\n",
        )
        hits = mod.collect_violations(tmp, check_meta=False)
        assert hits == [], hits


def test_example_config_pem_fails() -> None:
    """*.example config still scanned for HIGH_CONFIDENCE leak patterns."""
    mod = _load()
    with tempfile.TemporaryDirectory() as raw:
        tmp = Path(raw)
        _write(
            tmp / "trae-agent" / "onlineServiceJS" / "trae_config.yaml.example",
            "# example\n" + _pem_block(),
        )
        hits = mod.collect_violations(tmp, check_meta=False)
        assert any("pem-private-key" in h for h in hits), hits


def test_example_config_stripe_live_fails() -> None:
    mod = _load()
    with tempfile.TemporaryDirectory() as raw:
        tmp = Path(raw)
        _write(
            tmp / "trae-agent" / "onlineServiceJS" / "trae_config.yaml.example",
            f"stripe_key: {_stripe_live()}\n",
        )
        hits = mod.collect_violations(tmp, check_meta=False)
        assert any("stripe-live" in h for h in hits), hits


def test_example_json_aws_access_key_fails() -> None:
    mod = _load()
    with tempfile.TemporaryDirectory() as raw:
        tmp = Path(raw)
        _write(
            tmp / "trae-agent" / "onlineServiceJS" / "trae_config.json.example",
            '{"aws_access_key_id": "' + _aws_id() + '"}',
        )
        hits = mod.collect_violations(tmp, check_meta=False)
        assert any("aws-access-key" in h for h in hits), hits


def main() -> int:
    tests = [
        test_hardcoded_password_in_go_fails,
        test_getenv_assignment_passes,
        test_conf_yaml_secret_passes,
        test_pem_in_source_fails,
        test_pem_in_test_file_still_fails,
        test_placeholder_in_unit_test_passes,
        test_aws_access_key_fails,
        test_well_known_aws_example_passes,
        test_stripe_live_fails,
        test_github_pat_fails,
        test_waiver_comment_passes,
        test_third_party_skipped,
        test_gitlab_ce_skipped,
        test_placeholder_literal_in_prod_passes,
        test_missing_meta_fails,
        test_meta_present_empty_tree_passes,
        test_dotenv_on_disk_not_scanned,
        test_is_placeholder_helpers,
        test_vue_kebab_password_prop_passes,
        test_cjk_error_message_assignment_passes,
        test_example_config_placeholder_passes,
        test_example_config_pem_fails,
        test_example_config_stripe_live_fails,
        test_example_json_aws_access_key_fails,
    ]
    failed = 0
    for fn in tests:
        try:
            fn()
            print(f"ok  {fn.__name__}")
        except Exception as exc:  # noqa: BLE001 — self-test runner
            failed += 1
            print(f"FAIL {fn.__name__}: {exc}")
    if failed:
        print(f"{failed}/{len(tests)} failed")
        return 1
    print(f"{len(tests)}/{len(tests)} passed")
    return 0


if __name__ == "__main__":
    sys.exit(main())
