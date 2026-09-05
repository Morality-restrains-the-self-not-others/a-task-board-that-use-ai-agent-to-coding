"""Unit tests for check_fe_url_vs_gateway_routes.py (OPT-20260807-037)."""

from __future__ import annotations

import tempfile
import unittest
from pathlib import Path

from check_fe_url_vs_gateway_routes import (
    fe_normalize,
    is_covered,
    load_gateway_uris,
    scan_file,
    strip_query,
)

# 样例网关路由（指向真实 upstream）
COVER_URIS = [
    "/api/tenant/*/billing/*",
    "/api/tenant/*",
    "/api/billing/*",
    "/api/projects/*",
    "/api/system-admin/cloud/*",
    "/api/accounts/users/*",
    "/api/cloud/*",
]


class StripQueryTests(unittest.TestCase):
    def test_plain_query(self) -> None:
        self.assertEqual(strip_query("/api/tenant/t1/billing/orders/?page=1"), "/api/tenant/t1/billing/orders/")

    def test_query_with_interpolation(self) -> None:
        self.assertEqual(
            strip_query("/api/tenant/${tid}/billing/orders/?${params}"),
            "/api/tenant/${tid}/billing/orders/",
        )

    def test_ternary_question_inside_interp_is_not_split(self) -> None:
        self.assertEqual(
            strip_query("/api/x/${a ? 'y' : 'z'}/tail"),
            "/api/x/${a ? 'y' : 'z'}/tail",
        )


class FeNormalizeTests(unittest.TestCase):
    def test_param_to_star(self) -> None:
        self.assertEqual(fe_normalize("/api/tenant/${tid}/billing/orders/"), "/api/tenant/*/billing/orders/")

    def test_encode_uri_component(self) -> None:
        self.assertEqual(
            fe_normalize("/api/tenant/${encodeURIComponent(tid)}/billing/orders/"),
            "/api/tenant/*/billing/orders/",
        )

    def test_braced_param(self) -> None:
        self.assertEqual(fe_normalize("/api/tenant/{tid}/billing/units"), "/api/tenant/*/billing/units")

    def test_query_stripped(self) -> None:
        self.assertEqual(
            fe_normalize("/api/tenant/${tid}/billing/orders/?${params}"),
            "/api/tenant/*/billing/orders/",
        )


class IsCoveredTests(unittest.TestCase):
    def test_billing_tenant_direct_covered(self) -> None:
        self.assertTrue(is_covered("/api/tenant/${tid}/billing/orders/", COVER_URIS))

    def test_billing_deep_path_covered_by_wildcard(self) -> None:
        self.assertTrue(is_covered("/api/tenant/${tid}/billing/orders/${id}/pay/", COVER_URIS))

    def test_unregistered_prefix_flagged(self) -> None:
        # 假设前端引入全新前缀 /api/newmodule/*（网关无对应路由）→ 应阻断
        self.assertFalse(is_covered("/api/newmodule/${id}/thing/", COVER_URIS))

    def test_allowlist_orphan_page(self) -> None:
        self.assertTrue(is_covered("/api/system-admin/${uid}/cloud/server-images/", COVER_URIS))
        self.assertTrue(is_covered("/api/system-admin/${uid}/cloud/server-images/${id}/", COVER_URIS))

    def test_allowlist_form_default(self) -> None:
        self.assertTrue(is_covered("/api/token/derive", COVER_URIS))

    def test_non_api_ignored(self) -> None:
        self.assertTrue(is_covered("/gateway/ops/whatever", COVER_URIS))


class LoadGatewayUrisTests(unittest.TestCase):
    def test_real_routes_yaml(self) -> None:
        cover, fallback = load_gateway_uris()
        self.assertGreater(len(cover), 50)
        # 关键前缀必须存在（billing 事故修复后的直连路由）
        self.assertTrue(any("/api/tenant/*/billing/*" == u for u in cover))
        # null-upstream 兜底单独分类
        self.assertGreater(len(fallback), 0)
        self.assertTrue(all("/api/*" in u for u in fallback))


class ScanFileTests(unittest.TestCase):
    def _scan(self, src: str) -> list:
        with tempfile.TemporaryDirectory() as tmp:
            p = Path(tmp) / "x.vue"
            p.write_text(src, encoding="utf-8")
            hits = []
            scan_file(str(p), COVER_URIS, hits, "x.vue")
            return hits

    def test_real_fetch_uncovered_flagged(self) -> None:
        hits = self._scan('await apiFetch(`/api/newmodule/${id}/thing/`)')
        self.assertEqual(len(hits), 1)

    def test_real_fetch_covered_ok(self) -> None:
        self.assertEqual(self._scan('await apiFetch(`/api/tenant/${tid}/billing/orders/`)'), [])

    def test_comment_skipped(self) -> None:
        self.assertEqual(self._scan("// 测试注释：/api/newmodule/${id}/thing/"), [])

    def test_test_description_skipped(self) -> None:
        self.assertEqual(self._scan('it("挂载后调用 /api/newmodule/${id}/thing/", () => {})'), [])

    def test_replace_replacement_skipped(self) -> None:
        self.assertEqual(self._scan('const s = u.replace(/x/, "/api/newmodule/${id}/thing/")'), [])


if __name__ == "__main__":
    unittest.main()
