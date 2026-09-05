#!/usr/bin/env python3
"""Unit tests for check_copyright_ssot.py (OPT-20260824-013)."""

from __future__ import annotations

import tempfile
import unittest
from pathlib import Path

from check_copyright_ssot import (
    SSOT_COPYRIGHT,
    check_file,
    collect_violations,
    excluded,
    iter_readmes,
)


class ExcludedTests(unittest.TestCase):
    def setUp(self) -> None:
        self._tmp = tempfile.TemporaryDirectory()
        self.root = Path(self._tmp.name)

    def tearDown(self) -> None:
        self._tmp.cleanup()

    def test_third_party_excluded(self) -> None:
        p = self.root / "taskX" / "third_party" / "README.md"
        self.assertTrue(excluded(p, self.root))

    def test_trae_agent_excluded(self) -> None:
        p = self.root / "trae-agent" / "README.md"
        self.assertTrue(excluded(p, self.root))

    def test_git_service_excluded(self) -> None:
        p = self.root / "gitService" / "README.md"
        self.assertTrue(excluded(p, self.root))

    def test_sdk_excluded(self) -> None:
        p = self.root / "sdk" / "wechatpay-go" / "README.md"
        self.assertTrue(excluded(p, self.root))

    def test_archi_excluded(self) -> None:
        p = self.root / "docs" / "architecture" / "Archi" / "README.md"
        self.assertTrue(excluded(p, self.root))

    def test_first_party_not_excluded(self) -> None:
        p = self.root / "taskFE" / "README.md"
        self.assertFalse(excluded(p, self.root))


class CheckFileTests(unittest.TestCase):
    def setUp(self) -> None:
        self._tmp = tempfile.TemporaryDirectory()
        self.root = Path(self._tmp.name)

    def tearDown(self) -> None:
        self._tmp.cleanup()

    def _write(self, rel: str, content: str) -> Path:
        p = self.root / rel
        p.parent.mkdir(parents=True, exist_ok=True)
        p.write_text(content, encoding="utf-8")
        return p

    def test_matching_copyright_ok(self) -> None:
        p = self._write("README.md", f"# X\n\n{SSOT_COPYRIGHT}\n")
        self.assertEqual(check_file(p, self.root), [])

    def test_drift_year_fails(self) -> None:
        p = self._write("README.md", "# X\n\nCopyright (c) 2026 author@example.com\n")
        hits = check_file(p, self.root)
        self.assertEqual(len(hits), 1)
        self.assertIn("README.md:3", hits[0])

    def test_drift_owner_fails(self) -> None:
        p = self._write("README.md", "# X\n\nCopyright (c) 2025～2026 someone@else.com\n")
        hits = check_file(p, self.root)
        self.assertEqual(len(hits), 1)

    def test_multi_copyright_lines_all_checked(self) -> None:
        p = self._write(
            "README.md",
            f"# X\n\n{SSOT_COPYRIGHT}\n\nCopyright (c) 2025 other\n",
        )
        hits = check_file(p, self.root)
        self.assertEqual(len(hits), 1)

    def test_no_copyright_line_ok(self) -> None:
        p = self._write("README.md", "# X\n\nno copyright here\n")
        self.assertEqual(check_file(p, self.root), [])


class IterReadmesTests(unittest.TestCase):
    def setUp(self) -> None:
        self._tmp = tempfile.TemporaryDirectory()
        self.root = Path(self._tmp.name)

    def tearDown(self) -> None:
        self._tmp.cleanup()

    def test_skips_excluded_but_keeps_first_party(self) -> None:
        (self.root / "taskFE").mkdir(parents=True)
        (self.root / "trae-agent").mkdir(parents=True)
        (self.root / "taskFE" / "README.md").write_text("# a\n", encoding="utf-8")
        (self.root / "trae-agent" / "README.md").write_text("# b\n", encoding="utf-8")
        rels = [p.relative_to(self.root).as_posix() for p in iter_readmes(self.root)]
        self.assertEqual(rels, ["taskFE/README.md"])

    def test_collect_violations_reports_drift(self) -> None:
        (self.root / "taskFE").mkdir(parents=True)
        (self.root / "taskFE" / "README.md").write_text(
            "# a\n\nCopyright (c) 2024 old\n", encoding="utf-8"
        )
        hits = collect_violations(self.root)
        self.assertEqual(len(hits), 1)
        self.assertIn("taskFE/README.md", hits[0])


if __name__ == "__main__":
    unittest.main()
