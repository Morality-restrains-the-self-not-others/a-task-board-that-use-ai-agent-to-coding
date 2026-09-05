#!/usr/bin/env python3
"""Self-test for check_design_md (no pytest required)."""

from __future__ import annotations

import importlib.util
import sys
import tempfile
from pathlib import Path

ROOT = Path(__file__).resolve().parents[3]
CHECKER = ROOT / "db" / "scripts" / "ci" / "check_design_md.py"

MIN_DESIGN = """---
name: fixture-design
description: "fixture"
colors:
  primary: "#7241F2"
---

## Colors
primary is the brand accent.

## Typography
System sans.

## Components
Buttons use rounded-md.

## Layout
Navbar then sidebar.

## Do's and Don'ts
Do reuse tokens. Don't invent hex.
"""

MIN_CSS = """
:root {
  --color-primary: #7241F2;
}
"""

MIN_SKILL = """---
name: design-md
description: Use DESIGN.md as the visual SSOT for product UI.
---

# Design.md
Read DESIGN.md before styling product chrome.
"""


def _load():
    spec = importlib.util.spec_from_file_location("check_design_md", CHECKER)
    assert spec and spec.loader
    mod = importlib.util.module_from_spec(spec)
    sys.modules[spec.name] = mod
    spec.loader.exec_module(mod)
    return mod


def _tree(tmp: Path, design: str = MIN_DESIGN, css: str = MIN_CSS, skill: str = MIN_SKILL) -> None:
    (tmp / "DESIGN.md").write_text(design, encoding="utf-8")
    css_path = tmp / "taskFE" / "app" / "src" / "css"
    css_path.mkdir(parents=True)
    (css_path / "tailwind.css").write_text(css, encoding="utf-8")
    skill_dir = tmp / ".claude" / "skills" / "design-md"
    skill_dir.mkdir(parents=True)
    (skill_dir / "SKILL.md").write_text(skill, encoding="utf-8")
    readme = tmp / ".claude" / "skills" / "README.md"
    readme.write_text("| `design-md` | DESIGN.md visual baseline |\n", encoding="utf-8")


def test_missing_design_md_is_violation() -> None:
    mod = _load()
    with tempfile.TemporaryDirectory() as raw:
        tmp = Path(raw)
        _tree(tmp)
        (tmp / "DESIGN.md").unlink()
        hits = mod.collect_violations(tmp)
        assert any("DESIGN.md" in h and "missing" in h for h in hits), hits


def test_missing_required_heading_is_violation() -> None:
    mod = _load()
    with tempfile.TemporaryDirectory() as raw:
        tmp = Path(raw)
        broken = MIN_DESIGN.replace("## Typography\nSystem sans.\n", "")
        _tree(tmp, design=broken)
        hits = mod.collect_violations(tmp)
        assert any("Typography" in h for h in hits), hits


def test_primary_mismatch_is_violation() -> None:
    mod = _load()
    with tempfile.TemporaryDirectory() as raw:
        tmp = Path(raw)
        _tree(tmp, css=":root { --color-primary: #111111; }\n")
        hits = mod.collect_violations(tmp)
        assert any("primary" in h.lower() and "mismatch" in h.lower() for h in hits), hits


def test_missing_skill_is_violation() -> None:
    mod = _load()
    with tempfile.TemporaryDirectory() as raw:
        tmp = Path(raw)
        _tree(tmp)
        (tmp / ".claude" / "skills" / "design-md" / "SKILL.md").unlink()
        hits = mod.collect_violations(tmp)
        assert any("design-md" in h and "SKILL.md" in h for h in hits), hits


def test_good_tree_passes() -> None:
    mod = _load()
    with tempfile.TemporaryDirectory() as raw:
        tmp = Path(raw)
        _tree(tmp)
        hits = mod.collect_violations(tmp)
        assert hits == [], hits


def test_live_repo_passes() -> None:
    mod = _load()
    hits = mod.collect_violations(ROOT)
    assert hits == [], hits


if __name__ == "__main__":
    tests = [v for k, v in sorted(globals().items()) if k.startswith("test_")]
    for t in tests:
        t()
        print(f"PASS {t.__name__}")
    print(f"ok ({len(tests)} tests)")
