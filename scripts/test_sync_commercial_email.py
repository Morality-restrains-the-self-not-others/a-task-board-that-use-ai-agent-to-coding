#!/usr/bin/env python3
"""Unit tests for scripts/sync-commercial-email.py (OPT-20260824-087).

覆盖：SSOT 提取（README/COMMERCIAL 一致、不一致报错）、纯文本与 markdown 链接
格式替换、--check 漂移检测、--dry-run 不写盘、--repo 定向同步。
"""
from __future__ import annotations

import tempfile
import unittest
from pathlib import Path

import sync_commercial_email as sse


def _make_root() -> tuple[Path, Path]:
    root = Path(tempfile.mkdtemp(prefix="sync-email-"))
    sub = root / "taskAuth"
    sub.mkdir()
    (root / "README.md").write_text(
        "# root\n\n商务联系：contact@daydaymoney.com\n",
        encoding="utf-8",
    )
    (root / "COMMERCIAL.md").write_text(
        "# commercial\n\n1. 邮件联系 [contact@daydaymoney.com](mailto:contact@daydaymoney.com)，提供企业信息、使用场景；\n",
        encoding="utf-8",
    )
    (sub / "README.md").write_text(
        "# taskAuth\n\n商务联系：contact@daydaymoney.com\n",
        encoding="utf-8",
    )
    (sub / "COMMERCIAL.md").write_text(
        "# commercial\n\n1. 邮件联系 contact@daydaymoney.com，提供企业信息、使用场景；\n",
        encoding="utf-8",
    )
    return root, sub


class SSOTExtractionTests(unittest.TestCase):
    def test_extract_both(self) -> None:
        root, _ = _make_root()
        emails = sse.ssot_emails(root)
        self.assertEqual(emails["README"], "contact@daydaymoney.com")
        self.assertEqual(emails["COMMERCIAL"], "contact@daydaymoney.com")

    def test_missing_ssot_raises(self) -> None:
        root = Path(tempfile.mkdtemp(prefix="sync-email-missing-"))
        with self.assertRaises(ValueError):
            sse.ssot_emails(root)

    def test_conflicting_ssot(self) -> None:
        root = Path(tempfile.mkdtemp(prefix="sync-email-conflict-"))
        (root / "README.md").write_text("商务联系：a@b.com\n", encoding="utf-8")
        (root / "COMMERCIAL.md").write_text("1. 邮件联系 c@d.com，提供企业信息、使用场景；\n", encoding="utf-8")
        emails = sse.ssot_emails(root)
        self.assertNotEqual(emails["README"], emails["COMMERCIAL"])


class SyncFileTests(unittest.TestCase):
    def test_plain_text_commit_line(self) -> None:
        root = Path(tempfile.mkdtemp(prefix="sync-email-sync-"))
        p = root / "COMMERCIAL.md"
        p.write_text("1. 邮件联系 old@x.com，提供企业信息、使用场景；\n", encoding="utf-8")
        changed, _ = sse.sync_file(p, "new@y.com", dry_run=False)
        self.assertTrue(changed)
        self.assertIn("new@y.com", p.read_text(encoding="utf-8"))

    def test_markdown_link_line(self) -> None:
        root = Path(tempfile.mkdtemp(prefix="sync-email-md-"))
        p = root / "COMMERCIAL.md"
        p.write_text("1. 邮件联系 [old@x.com](mailto:old@x.com)，提供企业信息、使用场景；\n", encoding="utf-8")
        sse.sync_file(p, "new@y.com", dry_run=False)
        self.assertIn("[new@y.com](mailto:new@y.com)", p.read_text(encoding="utf-8"))

    def test_readme_biz_line(self) -> None:
        root = Path(tempfile.mkdtemp(prefix="sync-email-readme-"))
        p = root / "README.md"
        p.write_text("商务联系：old@x.com\n", encoding="utf-8")
        sse.sync_file(p, "new@y.com", dry_run=False)
        self.assertIn("商务联系：new@y.com", p.read_text(encoding="utf-8"))

    def test_dry_run_no_write(self) -> None:
        root = Path(tempfile.mkdtemp(prefix="sync-email-dry-"))
        p = root / "README.md"
        p.write_text("商务联系：old@x.com\n", encoding="utf-8")
        changed, _ = sse.sync_file(p, "new@y.com", dry_run=True)
        self.assertTrue(changed)
        self.assertIn("old@x.com", p.read_text(encoding="utf-8"))


class DriftTests(unittest.TestCase):
    def test_check_detects_drift(self) -> None:
        root, sub = _make_root()
        (sub / "COMMERCIAL.md").write_text("1. 邮件联系 drifted@x.com，提供企业信息、使用场景；\n", encoding="utf-8")
        rc = sse.main(["--root", str(root), "--check"])
        self.assertEqual(rc, 1)
        self.assertIn("drifted@x.com", (sub / "COMMERCIAL.md").read_text(encoding="utf-8"))

    def test_sync_fixes_drift(self) -> None:
        root, sub = _make_root()
        (sub / "COMMERCIAL.md").write_text("1. 邮件联系 drifted@x.com，提供企业信息、使用场景；\n", encoding="utf-8")
        rc = sse.main(["--root", str(root)])
        self.assertEqual(rc, 0)
        self.assertIn("contact@daydaymoney.com", (sub / "COMMERCIAL.md").read_text(encoding="utf-8"))

    def test_repo_target_only(self) -> None:
        root, sub = _make_root()
        other = root / "taskBill"
        other.mkdir()
        (other / "README.md").write_text("商务联系：drift2@x.com\n", encoding="utf-8")
        rc = sse.main(["--root", str(root), "--repo", "taskAuth"])
        self.assertEqual(rc, 0)
        self.assertIn("drift2@x.com", (other / "README.md").read_text(encoding="utf-8"))


if __name__ == "__main__":
    unittest.main()
