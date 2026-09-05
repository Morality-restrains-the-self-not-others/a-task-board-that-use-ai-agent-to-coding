"""ADR-0054: overlay_conf_file merges conf-local onto tracked conf YAML."""
from __future__ import annotations

import unittest
from pathlib import Path

from conf_local import overlay_conf_file


class TestOverlayConfFile(unittest.TestCase):
    def test_overlays_nested_rel(self) -> None:
        root = Path(self._tmp())
        conf = root / "conf" / "billing" / "paypal"
        loc = root / "conf-local" / "billing" / "paypal"
        conf.mkdir(parents=True)
        loc.mkdir(parents=True)
        (root / "conf" / "base.yaml").write_text("scheme: https\nbaseDomain: example.test\n", encoding="utf-8")
        (conf / "config.yaml").write_text('client_id: pub\nclient_secret: ""\n', encoding="utf-8")
        (loc / "config.yaml").write_text("client_secret: from-conf-local\n", encoding="utf-8")
        got = overlay_conf_file(conf / "config.yaml")
        self.assertEqual(got.get("client_id"), "pub")
        self.assertEqual(got.get("client_secret"), "from-conf-local")

    def test_missing_conf_local_keeps_tracked(self) -> None:
        root = Path(self._tmp())
        conf = root / "conf" / "infra" / "mysql"
        conf.mkdir(parents=True)
        (root / "conf" / "base.yaml").write_text("scheme: https\nbaseDomain: example.test\n", encoding="utf-8")
        (conf / "config.yaml").write_text("maxConnections: 12\n", encoding="utf-8")
        got = overlay_conf_file(conf / "config.yaml")
        self.assertEqual(got.get("maxConnections"), 12)

    def _tmp(self) -> str:
        import tempfile

        d = tempfile.mkdtemp()
        self.addCleanup(lambda: __import__("shutil").rmtree(d, ignore_errors=True))
        return d


if __name__ == "__main__":
    unittest.main(verbosity=2)
