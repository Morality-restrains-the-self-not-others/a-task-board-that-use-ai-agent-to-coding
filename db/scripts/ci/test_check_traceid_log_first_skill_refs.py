#!/usr/bin/env python3
"""Unit check for check_traceid_log_first_skill_refs.py helpers."""

from __future__ import annotations

import importlib.util
from pathlib import Path

HERE = Path(__file__).resolve().parent
MOD_PATH = HERE / "check_traceid_log_first_skill_refs.py"


def _load():
    spec = importlib.util.spec_from_file_location("check_traceid_log_first_skill_refs", MOD_PATH)
    mod = importlib.util.module_from_spec(spec)
    assert spec.loader is not None
    spec.loader.exec_module(mod)
    return mod


def test_skill_ok_accepts_path_ref() -> None:
    mod = _load()
    assert mod.skill_ok("see traceid-log-first-diagnosis.md")


def test_skill_ok_accepts_chinese_phrase() -> None:
    mod = _load()
    assert mod.skill_ok("有 data-traceId 先查日志再改代码")


def test_skill_ok_rejects_unrelated() -> None:
    mod = _load()
    assert not mod.skill_ok("just write more tests")


def test_main_passes_on_repo() -> None:
    mod = _load()
    assert mod.main() == 0


if __name__ == "__main__":
    test_skill_ok_accepts_path_ref()
    test_skill_ok_accepts_chinese_phrase()
    test_skill_ok_rejects_unrelated()
    test_main_passes_on_repo()
    print("OK test_check_traceid_log_first_skill_refs")
