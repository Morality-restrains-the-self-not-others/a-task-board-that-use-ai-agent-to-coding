#!/usr/bin/env python3
"""Unit tests for check_frontend_head_trae_service.py."""

from __future__ import annotations

import tempfile
import unittest
from pathlib import Path

from check_frontend_head_trae_service import check, extract_trae_service


class ExtractTests(unittest.TestCase):
    def test_extract_ok(self) -> None:
        html = '<head><meta name="trae-service" content="taskAiProvider" /></head>'
        self.assertEqual(extract_trae_service(html), "taskAiProvider")

    def test_extract_missing(self) -> None:
        self.assertIsNone(extract_trae_service("<head><title>x</title></head>"))


class CheckTests(unittest.TestCase):
    def test_check_pass_and_fail(self) -> None:
        with tempfile.TemporaryDirectory() as tmp:
            root = Path(tmp)
            (root / "db").mkdir()
            (root / "db" / "registry.yaml").write_text("services: {}\n", encoding="utf-8")
            page = root / "app" / "index.html"
            page.parent.mkdir(parents=True)
            page.write_text(
                '<!DOCTYPE html><html><head><meta name="trae-service" content="demo" /></head></html>',
                encoding="utf-8",
            )
            manifest = root / "manifest.yaml"
            manifest.write_text(
                "entries:\n  - path: app/index.html\n    service: demo\n",
                encoding="utf-8",
            )
            self.assertEqual(check(root, manifest), [])

            page.write_text("<!DOCTYPE html><html><head></head></html>", encoding="utf-8")
            errs = check(root, manifest)
            self.assertEqual(len(errs), 1)
            self.assertIn("missing", errs[0])


if __name__ == "__main__":
    unittest.main()
