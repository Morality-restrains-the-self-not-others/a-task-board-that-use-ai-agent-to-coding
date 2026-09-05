from __future__ import annotations

import sys
from pathlib import Path

SCRIPTS = Path(__file__).resolve().parent
sys.path.insert(0, str(SCRIPTS))
from generate_prometheus_from_runall import load_yaml  # noqa: E402


def test_load_yaml_overlays_conf_local(tmp_path: Path) -> None:
    conf = tmp_path / "conf"
    conf.mkdir()
    (conf / "base.yaml").write_text("scheme: https\nbaseDomain: example.test\n", encoding="utf-8")
    runall = conf / "runAll.yaml"
    runall.write_text("logging:\n  file_root: tracked\n", encoding="utf-8")
    local = tmp_path / "conf-local"
    local.mkdir()
    (local / "runAll.yaml").write_text("logging:\n  file_root: from-conf-local\n", encoding="utf-8")
    got = load_yaml(runall)
    assert got["logging"]["file_root"] == "from-conf-local"
