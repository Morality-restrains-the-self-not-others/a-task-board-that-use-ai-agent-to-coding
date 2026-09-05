#!/usr/bin/env python3
"""Smoke tests for check_archimate_full_inherits_views.

Run: python3 db/scripts/ci/test_check_archimate_full_inherits_views.py
"""

from __future__ import annotations

import sys
import tempfile
from pathlib import Path

ROOT = Path(__file__).resolve().parents[3]
sys.path.insert(0, str(ROOT / "db" / "scripts" / "ci"))

from check_archimate_full_inherits_views import (  # noqa: E402
    diagram_names,
    main,
    run,
)

MINIMAL_FULL = """<?xml version="1.0" encoding="UTF-8"?>
<archimate:model xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance"
  xmlns:archimate="http://www.archimatetool.com/archimate" name="t" id="m1">
  <folder name="Application" id="fa" type="application">
    <element xsi:type="archimate:ApplicationComponent" id="e1" name="A"/>
    <element xsi:type="archimate:ApplicationComponent" id="e2" name="B"/>
  </folder>
  <folder name="Relations" id="fr" type="relations">
    <element xsi:type="archimate:FlowRelationship" id="r1" source="e1" target="e2"/>
  </folder>
  <folder name="Views" id="fv" type="diagrams">
    {views}
  </folder>
</archimate:model>
"""


def _view(vid: str, name: str) -> str:
    return (
        f'<element xsi:type="archimate:ArchimateDiagramModel" '
        f'id="{vid}" name="{name}">'
        f'<child xsi:type="archimate:DiagramObject" id="do-{vid}" '
        f'archimateElement="e1"><bounds x="0" y="0" width="10" height="10"/>'
        f"</child></element>"
    )


def test_diagram_names_extracts() -> None:
    xml = MINIMAL_FULL.format(
        views=_view("v1", "topo-a") + _view("v2", "mig-a")
    )
    names = diagram_names(xml)
    assert "topo-a" in names and "mig-a" in names


def test_inherits_ok() -> None:
    with tempfile.TemporaryDirectory() as td:
        arch = Path(td)
        (arch / "archive").mkdir()
        prior_views = (
            _view("va", "old-topo")
            + _view("vb", "old-mig")
            + _view("vc", "older")
        )
        cur_views = prior_views + _view("vd", "new-topo") + _view("ve", "new-mig")
        (arch / "archive" / "v10-enterprise-landscape-20260101-1200-x.full.archimate").write_text(
            MINIMAL_FULL.format(views=prior_views), encoding="utf-8"
        )
        (arch / "v11-enterprise-landscape-20260102-1200-y.full.archimate").write_text(
            MINIMAL_FULL.format(views=cur_views), encoding="utf-8"
        )
        assert run(arch, strict_debt=False) == 0


def test_missing_inherited_view_fails() -> None:
    with tempfile.TemporaryDirectory() as td:
        arch = Path(td)
        (arch / "archive").mkdir()
        prior_views = (
            _view("va", "old-topo")
            + _view("vb", "old-mig")
            + _view("vc", "older")
        )
        # slice-only current (v200 outside debt range)
        cur_views = _view("vd", "new-topo") + _view("ve", "new-mig")
        (arch / "archive" / "v199-enterprise-landscape-20260101-1200-x.full.archimate").write_text(
            MINIMAL_FULL.format(views=prior_views), encoding="utf-8"
        )
        (arch / "v200-enterprise-landscape-20260102-1200-y.full.archimate").write_text(
            MINIMAL_FULL.format(views=cur_views), encoding="utf-8"
        )
        assert run(arch, strict_debt=False) == 1


def test_broken_inherit_fails_without_debt() -> None:
    """After OPT-011, no debt soft-warn — slice-only full must fail."""
    with tempfile.TemporaryDirectory() as td:
        arch = Path(td)
        (arch / "archive").mkdir()
        prior_views = (
            _view("va", "old-topo")
            + _view("vb", "old-mig")
            + _view("vc", "older")
        )
        cur_views = _view("vd", "slice-topo") + _view("ve", "slice-mig")
        (arch / "archive" / "v125-enterprise-landscape-20260101-1200-x.full.archimate").write_text(
            MINIMAL_FULL.format(views=prior_views), encoding="utf-8"
        )
        cur = arch / "v126-enterprise-landscape-20260102-1200-y.full.archimate"
        cur.write_text(MINIMAL_FULL.format(views=cur_views), encoding="utf-8")
        cur.with_name(cur.name.replace(".full.archimate", ".diff.archimate")).write_text(
            MINIMAL_FULL.format(views=cur_views), encoding="utf-8"
        )
        assert run(arch, strict_debt=False) == 1
        assert run(arch, strict_debt=True) == 1


def test_repo_default_exits_zero() -> None:
    # Live repo after OPT-011 backfill must pass default + strict-debt
    assert main([]) == 0
    assert main(["--strict-debt"]) == 0


def main_tests() -> int:
    tests = [
        test_diagram_names_extracts,
        test_inherits_ok,
        test_missing_inherited_view_fails,
        test_broken_inherit_fails_without_debt,
        test_repo_default_exits_zero,
    ]
    failed = 0
    for t in tests:
        try:
            t()
            print(f"PASS {t.__name__}")
        except Exception as e:  # noqa: BLE001
            failed += 1
            print(f"FAIL {t.__name__}: {e}")
    if failed:
        print(f"{failed}/{len(tests)} failed")
        return 1
    print(f"OK {len(tests)}/{len(tests)}")
    return 0


if __name__ == "__main__":
    raise SystemExit(main_tests())
