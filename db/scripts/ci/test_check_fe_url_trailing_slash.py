#!/usr/bin/env python3
"""Unit tests for check_fe_url_trailing_slash.py (OPT-20260807-036)."""

from __future__ import annotations

import tempfile
import unittest
from pathlib import Path

from check_fe_url_trailing_slash import (
    ROOT,
    extract_backend_leaves,
    scan_file,
    strip_query,
)


class StripQueryTests(unittest.TestCase):
    def test_plain_query(self) -> None:
        self.assertEqual(strip_query("/api/tenant/t1/billing/orders/?page=1"), "/api/tenant/t1/billing/orders/")

    def test_query_with_interpolation(self) -> None:
        self.assertEqual(
            strip_query("/api/tenant/${tid}/billing/orders/?${params}"),
            "/api/tenant/${tid}/billing/orders/",
        )

    def test_ternary_question_inside_interp_is_not_split(self) -> None:
        # ${...} 内的 ? 属于插值表达式，不是查询分隔符
        self.assertEqual(
            strip_query("/api/x/${a ? 'y' : 'z'}/tail"),
            "/api/x/${a ? 'y' : 'z'}/tail",
        )

    def test_fragment_stripped(self) -> None:
        self.assertEqual(strip_query("/api/x/#frag"), "/api/x/")

    def test_no_query(self) -> None:
        self.assertEqual(strip_query("/api/tenant/t1/billing/units/"), "/api/tenant/t1/billing/units/")


class ExtractLeavesTests(unittest.TestCase):
    def test_billing_leaves_present(self) -> None:
        leaves = extract_backend_leaves()
        suffixes = {s for _c, s in leaves}
        self.assertIn("/billing/units/", suffixes)
        self.assertIn("/billing/statistics/", suffixes)
        self.assertIn("/billing/orders/", suffixes)
        self.assertIn("/billing/transactions/", suffixes)

    def test_contains_precondition_captured(self) -> None:
        leaves = extract_backend_leaves()
        self.assertIn(("/billing/orders/", "/pay/"), leaves)
        self.assertIn(("/billing/orders/", "/cancel/"), leaves)

    def test_root_slash_excluded(self) -> None:
        suffixes = {s for _c, s in extract_backend_leaves()}
        self.assertNotIn("/", suffixes)


class ScanFileTests(unittest.TestCase):
    def _scan(self, src: str) -> list:
        with tempfile.TemporaryDirectory() as tmp:
            p = Path(tmp) / "x.js"
            p.write_text(src, encoding="utf-8")
            return scan_file(str(p), extract_backend_leaves())

    def test_missing_slash_flagged(self) -> None:
        hits = self._scan('apiFetch(`/api/tenant/${tid}/billing/units`)')
        self.assertEqual(len(hits), 1)
        self.assertIn("/billing/units/", hits[0][1])

    def test_with_slash_ok(self) -> None:
        self.assertEqual(self._scan('apiFetch(`/api/tenant/${tid}/billing/units/`)'), [])

    def test_query_tail_ok(self) -> None:
        self.assertEqual(self._scan('apiFetch(`/api/tenant/${tid}/billing/orders/?${params}`)'), [])

    def test_missing_slash_before_query_flagged(self) -> None:
        hits = self._scan('apiFetch(`/api/tenant/${tid}/billing/statistics?month=1`)')
        self.assertEqual(len(hits), 1)
        self.assertIn("/billing/statistics/", hits[0][1])

    def test_non_billing_api_not_flagged(self) -> None:
        self.assertEqual(self._scan('apiFetch(`/api/projects/tenant_id/${tid}`)'), [])

    def test_order_detail_with_id_not_flagged(self) -> None:
        # /billing/orders/{id} 由 Contains 匹配，无需尾斜杠
        self.assertEqual(self._scan('apiFetch(`/api/tenant/${tid}/billing/orders/${id}`)'), [])

    def test_contains_precondition_missing_not_flagged(self) -> None:
        # /pay/ 需前置 /billing/orders/；无该前缀不命中
        self.assertEqual(self._scan('apiFetch(`/api/tenant/${tid}/pay`)'), [])


if __name__ == "__main__":
    unittest.main()
