#!/usr/bin/env python3
"""Self-test for check_nfr_idempotency_table (no pytest required)."""

from __future__ import annotations

import importlib.util
import sys
from pathlib import Path

ROOT = Path(__file__).resolve().parents[3]
CHECKER = ROOT / "db" / "scripts" / "ci" / "check_nfr_idempotency_table.py"
SKILL = ROOT / ".claude" / "skills" / "5-nfr" / "SKILL.md"
REF = ROOT / ".claude" / "skills" / "5-nfr" / "references" / "idempotency.md"


def _load():
    spec = importlib.util.spec_from_file_location("check_nfr_idempotency_table", CHECKER)
    assert spec and spec.loader
    mod = importlib.util.module_from_spec(spec)
    sys.modules[spec.name] = mod
    spec.loader.exec_module(mod)
    return mod


def test_section_regex_detects_present_and_absent() -> None:
    mod = _load()
    good = "# Title\n\n## 幂等性审视\n\n| 路径 | 幂等键 | 说明 |\n"
    bad = "# Title\n\n## 其他\n\nno idempotency table here\n"
    assert mod.SECTION_RE.search(good)
    assert not mod.SECTION_RE.search(bad)


def test_scan_finds_missing() -> None:
    import tempfile

    mod = _load()
    with tempfile.TemporaryDirectory() as d:
        plans = Path(d) / "plans"
        plans.mkdir()
        (plans / "2026-08-01-demo-nfr-clarification.md").write_text("# X\n\n## 其他\n", encoding="utf-8")
        old = mod.PLANS
        mod.PLANS = plans
        try:
            missing = mod.scan()
        finally:
            mod.PLANS = old
        assert len(missing) == 1
        assert "demo-nfr-clarification" in missing[0].name


def test_strict_files_flag_fails_on_missing() -> None:
    import tempfile

    mod = _load()
    with tempfile.TemporaryDirectory() as d:
        tmp = Path(d)
        (tmp / "new-nfr-clarification.md").write_text("# New\n\n## 无幂等表\n", encoding="utf-8")
        old_root = mod.ROOT
        mod.ROOT = tmp
        try:
            rc = mod.main(["--strict", "--files", "new-nfr-clarification.md"])
        finally:
            mod.ROOT = old_root
        assert rc == 1


def test_warn_mode_returns_zero_on_missing() -> None:
    import tempfile

    mod = _load()
    with tempfile.TemporaryDirectory() as d:
        tmp = Path(d)
        (tmp / "old-nfr-clarification.md").write_text("# Old\n\n无表\n", encoding="utf-8")
        old_root = mod.ROOT
        mod.ROOT = tmp
        try:
            rc = mod.main(["--files", "old-nfr-clarification.md"])
        finally:
            mod.ROOT = old_root
        assert rc == 0


def test_skill_hard_gate_and_reference_exist() -> None:
    skill = SKILL.read_text(encoding="utf-8")
    assert "幂等性强制审视" in skill
    assert "幂等性审视" in skill
    assert REF.is_file()
    ref = REF.read_text(encoding="utf-8")
    assert "业务重复边界" in ref
    assert "company_id" in ref
    assert "user_id" in ref


if __name__ == "__main__":
    tests = [v for k, v in sorted(globals().items()) if k.startswith("test_")]
    for t in tests:
        t()
        print(f"PASS {t.__name__}")
    print(f"ok ({len(tests)} tests)")
