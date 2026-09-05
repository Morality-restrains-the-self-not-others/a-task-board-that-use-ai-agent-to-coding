from __future__ import annotations

import sys
from pathlib import Path

MYSQL_DIR = Path(__file__).resolve().parent
sys.path.insert(0, str(MYSQL_DIR))
from load_mysql_conf import load_mysql_conf  # noqa: E402


def test_load_mysql_conf_overlays_conf_local(tmp_path: Path) -> None:
    conf = tmp_path / "conf" / "infra" / "mysql"
    conf.mkdir(parents=True)
    (tmp_path / "conf" / "base.yaml").write_text("scheme: https\nbaseDomain: example.test\n", encoding="utf-8")
    (conf / "config.yaml").write_text("maxConnections: 100\n", encoding="utf-8")
    local = tmp_path / "conf-local" / "infra" / "mysql"
    local.mkdir(parents=True)
    (local / "config.yaml").write_text("maxConnections: 777\n", encoding="utf-8")
    got = load_mysql_conf(str(conf / "config.yaml"))
    assert got["maxConnections"] == 777


def test_load_mysql_conf_keeps_tracked_without_overlay(tmp_path: Path) -> None:
    conf = tmp_path / "conf" / "infra" / "mysql"
    conf.mkdir(parents=True)
    (tmp_path / "conf" / "base.yaml").write_text("scheme: https\nbaseDomain: example.test\n", encoding="utf-8")
    (conf / "config.yaml").write_text("maxConnections: 100\n", encoding="utf-8")
    got = load_mysql_conf(str(conf / "config.yaml"))
    assert got["maxConnections"] == 100
