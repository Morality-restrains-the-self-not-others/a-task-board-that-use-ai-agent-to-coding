#!/usr/bin/env python3
"""Self-test for check_frontend_button_anti_replay (no pytest required)."""

from __future__ import annotations

import importlib.util
import sys
import tempfile
from pathlib import Path

ROOT = Path(__file__).resolve().parents[3]
CHECKER = ROOT / "db" / "scripts" / "ci" / "check_frontend_button_anti_replay.py"
CONSTRAINT = ROOT / ".ai" / "01_project_constraints" / "57_frontend_button_anti_replay.md"
ADR = ROOT / "docs" / "adr" / "0020-frontend-button-anti-replay.md"
SKILL = ROOT / ".claude" / "skills" / "5-nfr" / "SKILL.md"


def _load():
    spec = importlib.util.spec_from_file_location("check_frontend_button_anti_replay", CHECKER)
    assert spec and spec.loader
    mod = importlib.util.module_from_spec(spec)
    sys.modules[spec.name] = mod
    spec.loader.exec_module(mod)
    return mod


def test_unguarded_vue_detected() -> None:
    mod = _load()
    bad = (
        "<template><button @click=\"save\">Save</button></template>\n"
        "<script setup>\n"
        "apiFetch('/x', { method: 'POST' })\n"
        "</script>\n"
    )
    good = (
        "<template><button @click=\"save\">Save</button></template>\n"
        "<script setup>\n"
        "import { createClickGuard } from '../utils/clickGuard.js'\n"
        "apiFetch('/x', { method: 'POST', headers: { 'Idempotency-Key': 'k' } })\n"
        "</script>\n"
    )
    assert mod.vue_mutating_click_unguarded(bad)
    assert not mod.vue_mutating_click_unguarded(good)
    ui = "<template><button @click=\"toggle\">Tab</button></template>\n<script setup></script>\n"
    assert not mod.vue_mutating_click_unguarded(ui)


def test_strict_files_blocks_unguarded() -> None:
    mod = _load()
    with tempfile.TemporaryDirectory() as d:
        root = Path(d)
        vue = root / "Bad.vue"
        vue.write_text(
            "<template><button @click=\"pay\">Pay</button></template>\n"
            "<script setup>fetch('/p', { method: 'POST' })</script>\n",
            encoding="utf-8",
        )
        hits = mod.scan_vue_files(root, [vue])
        assert hits and "mutating @click" in hits[0]


def test_repo_artifacts_present() -> None:
    mod = _load()
    hits = mod.missing_artifacts(ROOT)
    assert hits == [], hits


def test_main_ok_on_repo() -> None:
    mod = _load()
    assert mod.main([]) == 0


def test_strict_order_create_ok() -> None:
    mod = _load()
    rc = mod.main(["--strict", "--files", "taskFE/app/src/views/OrderCreate.vue"])
    assert rc == 0


def test_resolve_absolute_vue_path() -> None:
    mod = _load()
    vue = ROOT / "taskFE" / "app" / "src" / "views" / "OrderCreate.vue"
    targets = mod.resolve_file_patterns(ROOT, str(vue))
    assert targets == [vue.resolve()]
    rc = mod.main(["--strict", "--files", str(vue)])
    assert rc == 0


def test_constraint_and_adr_exist() -> None:
    text = CONSTRAINT.read_text(encoding="utf-8")
    assert "Idempotency-Key" in text
    assert "createClickGuard" in text
    assert "Anti-Replay-OK" in text
    adr = ADR.read_text(encoding="utf-8")
    assert "ADR-0020" in adr or "前端副作用按钮" in adr
    skill = SKILL.read_text(encoding="utf-8")
    assert "前端按钮防重放" in skill or "57_frontend_button_anti_replay" in skill


if __name__ == "__main__":
    tests = [v for k, v in sorted(globals().items()) if k.startswith("test_")]
    for t in tests:
        t()
        print(f"PASS {t.__name__}")
    print(f"ok ({len(tests)} tests)")
