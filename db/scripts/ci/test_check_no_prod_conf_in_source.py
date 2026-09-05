#!/usr/bin/env python3
"""Self-test for check_no_prod_conf_in_source (no pytest required)."""

from __future__ import annotations

import importlib.util
import sys
import tempfile
from pathlib import Path

ROOT = Path(__file__).resolve().parents[3]
CHECKER = ROOT / "db" / "scripts" / "ci" / "check_no_prod_conf_in_source.py"


def _load():
    spec = importlib.util.spec_from_file_location("check_no_prod_conf_in_source", CHECKER)
    assert spec and spec.loader
    mod = importlib.util.module_from_spec(spec)
    sys.modules[spec.name] = mod
    spec.loader.exec_module(mod)
    return mod


def _write(path: Path, text: str) -> None:
    path.parent.mkdir(parents=True, exist_ok=True)
    path.write_text(text, encoding="utf-8")


def test_config_local_in_service_fails() -> None:
    mod = _load()
    with tempfile.TemporaryDirectory() as raw:
        tmp = Path(raw)
        _write(tmp / "taskAuth" / "config.local.yaml", "password: hunter2-prod\n")
        hits = mod.collect_violations(tmp)
        assert any("config.local.yaml" in h for h in hits), hits


def test_config_local_in_root_conf_skipped() -> None:
    mod = _load()
    with tempfile.TemporaryDirectory() as raw:
        tmp = Path(raw)
        _write(tmp / "conf" / "auth" / "task-auth" / "config.local.yaml", "password: hunter2-prod\n")
        hits = mod.collect_violations(tmp)
        assert hits == [], hits


def test_example_placeholder_password_ok() -> None:
    mod = _load()
    with tempfile.TemporaryDirectory() as raw:
        tmp = Path(raw)
        _write(tmp / "conf.example" / "app.yaml", "password: REPLACE_WITH_SECRET\n")
        hits = mod.collect_violations(tmp)
        assert hits == [], hits


def test_example_real_password_fails() -> None:
    mod = _load()
    with tempfile.TemporaryDirectory() as raw:
        tmp = Path(raw)
        _write(tmp / "conf.example" / "app.yaml", "password: hunter2-prod-credential\n")
        hits = mod.collect_violations(tmp)
        assert any("conf.example" in h for h in hits), hits


def main() -> int:
    tests = [
        test_config_local_in_service_fails,
        test_config_local_in_root_conf_skipped,
        test_example_placeholder_password_ok,
        test_example_real_password_fails,
    ]
    failed = 0
    for fn in tests:
        try:
            fn()
            print(f"ok {fn.__name__}")
        except Exception as exc:
            failed += 1
            print(f"FAIL {fn.__name__}: {exc}")
    return 1 if failed else 0


if __name__ == "__main__":
    sys.exit(main())
