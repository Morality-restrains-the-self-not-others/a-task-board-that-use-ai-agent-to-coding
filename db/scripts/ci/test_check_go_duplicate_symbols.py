#!/usr/bin/env python3
"""Smoke tests for check_go_duplicate_symbols.py."""

from __future__ import annotations

import importlib.util
import subprocess
import sys
import tempfile
from pathlib import Path

ROOT = Path(__file__).resolve().parents[3]
CHECKER = ROOT / "db" / "scripts" / "ci" / "check_go_duplicate_symbols.py"


def _load():
    spec = importlib.util.spec_from_file_location("check_go_duplicate_symbols", CHECKER)
    assert spec and spec.loader
    mod = importlib.util.module_from_spec(spec)
    sys.modules[spec.name] = mod
    spec.loader.exec_module(mod)
    return mod


def test_detects_cross_file_duplicate_func() -> None:
    mod = _load()
    with tempfile.TemporaryDirectory() as tmp:
        d = Path(tmp)
        (d / "a.go").write_text("package main\n\nfunc Helper() {}\n", encoding="utf-8")
        (d / "b.go").write_text("package main\n\nfunc Helper() {}\n", encoding="utf-8")
        errs = mod.check_dir(d)
        assert any("Helper" in e for e in errs), errs


def test_allows_methods_and_init() -> None:
    mod = _load()
    with tempfile.TemporaryDirectory() as tmp:
        d = Path(tmp)
        (d / "a.go").write_text(
            "package main\n\ntype T struct{}\nfunc (t *T) Helper() {}\nfunc init() {}\n",
            encoding="utf-8",
        )
        (d / "b.go").write_text(
            "package main\n\nfunc (t T) Helper() {}\nfunc init() {}\n",
            encoding="utf-8",
        )
        errs = mod.check_dir(d)
        assert errs == [], errs


def test_cli_on_task_cloud_src() -> None:
    src = ROOT / "taskCloudService" / "src"
    completed = subprocess.run(
        [sys.executable, str(CHECKER), str(src)],
        cwd=ROOT,
        capture_output=True,
        text=True,
        check=False,
    )
    out = (completed.stdout or "") + (completed.stderr or "")
    assert completed.returncode == 0, out
    assert "OK duplicate-symbols" in out, out


if __name__ == "__main__":
    test_detects_cross_file_duplicate_func()
    test_allows_methods_and_init()
    test_cli_on_task_cloud_src()
    print("OK test_check_go_duplicate_symbols")
