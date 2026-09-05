"""Unit tests for build_service_url_api_catalog.py."""

from __future__ import annotations

import json
import tempfile
import unittest
from pathlib import Path

from build_service_url_api_catalog import (
    DASHBOARD_UID,
    classify_api,
    classify_fe,
    load_fe_pages,
    load_gateway_routes,
    load_ownership,
    match_owner,
    render_grafana_dashboard,
    render_markdown,
)

ROUTES_YAML = """
version: "1"
upstreams:
  taskAuth:
    host: 127.0.0.1
    port: 8003
  null-upstream:
    host: 127.0.0.1
    port: 1
routes:
  - id: auth-login
    uri: /api/auth/login/
    upstream: taskAuth
    auth_mode: public
  - id: billing
    uris:
      - /api/tenant/*/billing/*
    upstream: taskBill
    auth_mode: token
  - id: swagger
    uri: /api/swagger/
    upstream: taskAuth
    auth_mode: public
  - id: internal-kyc
    uri: /api/internal/kyc/*
    upstream: taskAuth
    auth_mode: machine
  - id: deny-internal
    uri: /api/internal/*
    auth_mode: deny
  - id: orphan
    uri: /api/legacy-orphan/*
    upstream: null-upstream
    auth_mode: public
"""

OWN_YAML = """
route_prefixes:
  - prefix: /api/auth/
    target_owner: task-auth
    status: go
  - prefix: /api/tenant/*/billing/
    target_owner: task-bill
    status: go
"""

FE_JS = """
export const publicRoutes = [
  { path: '/login/', name: 'login' },
  { path: '/faq/', name: 'faq' },
]
"""


class ClassifyApiTests(unittest.TestCase):
    def test_login_is_core(self) -> None:
        role, stream = classify_api("/api/auth/login/", "taskAuth", "public")
        self.assertEqual(role, "core")
        self.assertEqual(stream, "user-auth")

    def test_billing_is_core(self) -> None:
        role, stream = classify_api("/api/tenant/*/billing/*", "taskBill", "token")
        self.assertEqual(role, "core")
        self.assertEqual(stream, "billing")

    def test_system_admin_orders_is_core_billing(self) -> None:
        for uri in ("/api/system-admin/orders", "/api/system-admin/orders/", "/api/system-admin/orders/*"):
            role, stream = classify_api(uri, "taskBill", "token")
            self.assertEqual(role, "core", uri)
            self.assertEqual(stream, "billing", uri)

    def test_system_admin_profit_sharing_is_core_billing(self) -> None:
        role, stream = classify_api("/api/system-admin/profit-sharing/", "taskBill", "token")
        self.assertEqual(role, "core")
        self.assertEqual(stream, "billing")

    def test_public_pricing_is_core_billing(self) -> None:
        role, stream = classify_api("/api/public/product-pricing", "taskBill", "public")
        self.assertEqual(role, "core")
        self.assertEqual(stream, "billing")

    def test_system_admin_cloud_is_core_compute(self) -> None:
        role, stream = classify_api("/api/system-admin/cloud/*", "taskCloudService", "token")
        self.assertEqual(role, "core")
        self.assertEqual(stream, "compute")

    def test_sub_token_providers_is_core_compute(self) -> None:
        role, stream = classify_api("/api/system-admin/sub-token-providers/", "taskCloudService", "token")
        self.assertEqual(role, "core")
        self.assertEqual(stream, "compute")

    def test_referral_performance_is_core_referral(self) -> None:
        role, stream = classify_api("/api/system-admin/users/*/referral-performance/", "taskReferral", "token")
        self.assertEqual(role, "core")
        self.assertEqual(stream, "referral")

    def test_referral_stats_is_core_referral(self) -> None:
        role, stream = classify_api("/api/user/*/profile/referral-stats", "taskReferral", "token")
        self.assertEqual(role, "core")
        self.assertEqual(stream, "referral")

    def test_swagger_is_ops(self) -> None:
        role, stream = classify_api("/api/swagger/", "taskAuth", "public")
        self.assertEqual(role, "ops")
        self.assertIsNone(stream)

    def test_machine_internal_is_internal(self) -> None:
        role, _ = classify_api("/api/internal/kyc/*", "taskAuth", "machine")
        self.assertEqual(role, "internal")

    def test_deny_is_orphaned(self) -> None:
        role, _ = classify_api("/api/internal/*", None, "deny")
        self.assertEqual(role, "orphaned")

    def test_null_upstream_is_orphaned(self) -> None:
        role, _ = classify_api("/api/legacy-orphan/*", "null-upstream", "public")
        self.assertEqual(role, "orphaned")

    def test_unlisted_prefix_is_supporting(self) -> None:
        role, stream = classify_api("/api/ai-provider/marketplace/", "taskAiProvider", "token")
        self.assertEqual(role, "supporting")
        self.assertIsNone(stream)

    def test_spa_catchall_is_supporting(self) -> None:
        role, stream = classify_api("/*", "taskFE", "public")
        self.assertEqual(role, "supporting")
        self.assertIsNone(stream)


class ClassifyFeTests(unittest.TestCase):
    def test_login_page_is_core(self) -> None:
        self.assertEqual(classify_fe("/login/", "login")[0], "core")

    def test_faq_is_supporting(self) -> None:
        self.assertEqual(classify_fe("/faq/", "faq")[0], "supporting")

    def test_work_panel_is_core(self) -> None:
        role, stream = classify_fe("/tenant/:tenant/workspace/:id/work-panel/", "work_panel")
        self.assertEqual(role, "core")
        self.assertEqual(stream, "task")


class ParseTests(unittest.TestCase):
    def test_load_gateway_routes(self) -> None:
        with tempfile.TemporaryDirectory() as tmp:
            p = Path(tmp) / "routes.yaml"
            p.write_text(ROUTES_YAML, encoding="utf-8")
            rows = load_gateway_routes(p)
        ids = {r["id"] for r in rows}
        self.assertIn("auth-login", ids)
        self.assertIn("billing", ids)
        billing = next(r for r in rows if r["id"] == "billing")
        self.assertEqual(billing["uris"], ["/api/tenant/*/billing/*"])
        self.assertEqual(billing["upstream"], "taskBill")

    def test_load_ownership_and_match(self) -> None:
        with tempfile.TemporaryDirectory() as tmp:
            p = Path(tmp) / "own.yaml"
            p.write_text(OWN_YAML, encoding="utf-8")
            entries = load_ownership(p)
        owner = match_owner("/api/auth/login/", entries)
        self.assertEqual(owner["target_owner"], "task-auth")
        billing = match_owner("/api/tenant/*/billing/orders/", entries)
        self.assertEqual(billing["target_owner"], "task-bill")

    def test_load_fe_pages(self) -> None:
        with tempfile.TemporaryDirectory() as tmp:
            p = Path(tmp) / "publicRoutes.js"
            p.write_text(FE_JS, encoding="utf-8")
            pages = load_fe_pages([p])
        paths = {x["path"] for x in pages}
        self.assertIn("/login/", paths)
        self.assertIn("/faq/", paths)


class RenderTests(unittest.TestCase):
    def test_markdown_lists_core_and_orphaned(self) -> None:
        catalog = {
            "generated_from": ["routes.yaml"],
            "decision": "grafana-is-view-not-ssot",
            "api_routes": [
                {
                    "id": "auth-login",
                    "uri": "/api/auth/login/",
                    "upstream": "taskAuth",
                    "auth_mode": "public",
                    "owner": "task-auth",
                    "role": "core",
                    "value_stream": "user-auth",
                },
                {
                    "id": "orphan",
                    "uri": "/api/legacy-orphan/*",
                    "upstream": "null-upstream",
                    "auth_mode": "public",
                    "owner": "",
                    "role": "orphaned",
                    "value_stream": None,
                },
            ],
            "fe_pages": [
                {
                    "path": "/login/",
                    "name": "login",
                    "source": "publicRoutes.js",
                    "role": "core",
                    "value_stream": "user-auth",
                }
            ],
            "by_service": {"taskAuth": {"core": 1, "supporting": 0, "ops": 0, "internal": 0, "orphaned": 0}},
        }
        md = render_markdown(catalog)
        self.assertIn("user-auth", md)
        self.assertIn("/api/auth/login/", md)
        self.assertIn("orphaned", md)
        self.assertIn("/login/", md)

    def test_grafana_dashboard_uid_and_service_map(self) -> None:
        catalog = {
            "api_routes": [],
            "fe_pages": [],
            "by_service": {},
            "summary": {"api_routes": 0, "fe_pages": 0, "core_api": 0},
        }
        dash = render_grafana_dashboard(catalog)
        self.assertEqual(dash["uid"], DASHBOARD_UID)
        types = {p.get("type") for p in dash["panels"]}
        self.assertIn("nodeGraph", types)
        self.assertIn("text", types)
        self.assertIn("table", types)
        self.assertTrue(any("tempo" in json.dumps(p) for p in dash["panels"]))


if __name__ == "__main__":
    unittest.main()
