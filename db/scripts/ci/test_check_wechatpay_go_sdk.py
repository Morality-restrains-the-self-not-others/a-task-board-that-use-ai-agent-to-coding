#!/usr/bin/env python3
"""Self-test for check_wechatpay_go_sdk (no pytest required)."""

from __future__ import annotations

import importlib.util
import sys
import tempfile
from pathlib import Path

ROOT = Path(__file__).resolve().parents[3]
CHECKER = ROOT / "db" / "scripts" / "ci" / "check_wechatpay_go_sdk.py"


def _load():
    spec = importlib.util.spec_from_file_location("check_wechatpay_go_sdk", CHECKER)
    assert spec and spec.loader
    mod = importlib.util.module_from_spec(spec)
    sys.modules[spec.name] = mod
    spec.loader.exec_module(mod)
    return mod


def _write(path: Path, text: str) -> None:
    path.parent.mkdir(parents=True, exist_ok=True)
    path.write_text(text, encoding="utf-8")


def _sdk_tree(tmp: Path) -> None:
    _write(
        tmp / "sdk" / "wechatpay-go" / "go.mod",
        "module github.com/wechatpay-apiv3/wechatpay-go\n\ngo 1.16\n",
    )


def _meta(tmp: Path) -> None:
    for rel in (
        ".ai/01_project_constraints/59_wechatpay_go_sdk.md",
        ".cursor/rules/wechatpay-go-sdk.mdc",
        "docs/adr/0032-wechatpay-go-sdk.md",
    ):
        _write(tmp / rel, "# meta\n")


def _good_service(tmp: Path) -> None:
    _write(
        tmp / "taskBill" / "go.mod",
        "module taskBill\n\n"
        "require github.com/wechatpay-apiv3/wechatpay-go v0.2.21\n\n"
        "replace github.com/wechatpay-apiv3/wechatpay-go => ../sdk/wechatpay-go\n",
    )
    _write(
        tmp / "taskBill" / "src" / "wechat_pay.go",
        'package main\n\nimport "github.com/wechatpay-apiv3/wechatpay-go/core"\n',
    )


def test_require_without_replace_fails() -> None:
    mod = _load()
    with tempfile.TemporaryDirectory() as raw:
        tmp = Path(raw)
        _sdk_tree(tmp)
        _write(
            tmp / "taskBill" / "go.mod",
            "module taskBill\n\n"
            "require github.com/wechatpay-apiv3/wechatpay-go v0.2.21\n",
        )
        hits = mod.collect_violations(tmp, check_meta=False)
        assert any("missing replace" in h for h in hits), hits


def test_replace_to_sdk_passes() -> None:
    mod = _load()
    with tempfile.TemporaryDirectory() as raw:
        tmp = Path(raw)
        _sdk_tree(tmp)
        _meta(tmp)
        _good_service(tmp)
        hits = mod.collect_violations(tmp, check_meta=True)
        assert hits == [], hits


def test_replace_wrong_path_fails() -> None:
    mod = _load()
    with tempfile.TemporaryDirectory() as raw:
        tmp = Path(raw)
        _sdk_tree(tmp)
        _write(
            tmp / "taskBill" / "go.mod",
            "module taskBill\n\n"
            "require github.com/wechatpay-apiv3/wechatpay-go v0.2.21\n\n"
            "replace github.com/wechatpay-apiv3/wechatpay-go => ../vendor/wechatpay-go\n",
        )
        hits = mod.collect_violations(tmp, check_meta=False)
        assert any("must resolve to sdk/wechatpay-go" in h for h in hits), hits


def test_forbidden_import_fails() -> None:
    mod = _load()
    with tempfile.TemporaryDirectory() as raw:
        tmp = Path(raw)
        _sdk_tree(tmp)
        _good_service(tmp)
        _write(
            tmp / "taskBill" / "src" / "alt.go",
            'package main\n\nimport "github.com/go-pay/gopay/wechat/v3"\n',
        )
        hits = mod.collect_violations(tmp, check_meta=False)
        assert any("forbidden WeChat Pay SDK import" in h for h in hits), hits


def test_missing_sdk_dir_fails() -> None:
    mod = _load()
    with tempfile.TemporaryDirectory() as raw:
        tmp = Path(raw)
        hits = mod.collect_violations(tmp, check_meta=False)
        assert any("missing vendored WeChat Pay Go SDK" in h for h in hits), hits


def test_replace_helper_parses_block() -> None:
    mod = _load()
    text = (
        "module taskBill\n\n"
        "require (\n"
        "\tgithub.com/wechatpay-apiv3/wechatpay-go v0.2.21\n"
        ")\n\n"
        "replace (\n"
        "\tgithub.com/wechatpay-apiv3/wechatpay-go => ../sdk/wechatpay-go\n"
        ")\n"
    )
    assert mod.go_mod_requires_wechatpay(text)
    assert mod.go_mod_replace_target(text) == "../sdk/wechatpay-go"


def main() -> int:
    tests = [
        test_require_without_replace_fails,
        test_replace_to_sdk_passes,
        test_replace_wrong_path_fails,
        test_forbidden_import_fails,
        test_missing_sdk_dir_fails,
        test_replace_helper_parses_block,
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
