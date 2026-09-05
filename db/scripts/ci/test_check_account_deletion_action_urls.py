#!/usr/bin/env python3
"""Unit tests for check_account_deletion_action_urls.py (OPT-20260820-005)."""

from __future__ import annotations

import unittest

from check_account_deletion_action_urls import (
    check_all,
    collect_action_url_patterns,
    collect_router_paths,
    go_url_sample,
    route_to_regex,
)


class RouteToRegexTests(unittest.TestCase):
    def test_param_segment_becomes_any_segment(self) -> None:
        self.assertEqual(route_to_regex("/tenant/:tenant/billing/"), "^/tenant/[^/]+/billing/$")

    def test_root_slash(self) -> None:
        self.assertEqual(route_to_regex("/"), "^/$")

    def test_multiple_params(self) -> None:
        self.assertEqual(
            route_to_regex("/tenant/:tenant/workspace/:wid/task-detail/:tid/"),
            "^/tenant/[^/]+/workspace/[^/]+/task\\-detail/[^/]+/$",
        )


class GoUrlSampleTests(unittest.TestCase):
    def test_string_and_int_placeholders(self) -> None:
        self.assertEqual(go_url_sample("/tenant/%s/billing/orders/%d/"), "/tenant/x/billing/orders/1/")

    def test_literal_path_untouched(self) -> None:
        self.assertEqual(go_url_sample("/profile/"), "/profile/")


class RouterCollectionTests(unittest.TestCase):
    def test_registered_routes_present(self) -> None:
        registered, aliases = collect_router_paths()
        self.assertIn("/tenant/:tenant/billing/", registered)
        self.assertIn("/tenant/:tenant/settings/gitlab-connection/", registered)
        self.assertIn("/tenant/:tenant/people/manage/", registered)
        self.assertIn("/tenant/:tenant/workspace/:workspaceId/task-detail/:taskId/", registered)
        self.assertIn("/profile/", registered)

    def test_redirect_aliases_detected(self) -> None:
        registered, aliases = collect_router_paths()
        self.assertNotIn("/tenant/:tenant/billing/gitlab-resources/", registered)
        self.assertIn("/tenant/:tenant/billing/gitlab-resources/", aliases)
        self.assertIn("/tenant/:tenant/settings/members/", aliases)
        self.assertIn("/tenant/:tenant/workspace/:workspaceId/task/:taskId/", aliases)


class ActionUrlPatternCollectionTests(unittest.TestCase):
    def test_go_patterns_collected(self) -> None:
        patterns = [p for _f, p in collect_action_url_patterns()]
        self.assertGreaterEqual(len(patterns), 7)
        self.assertIn("/tenant/%s/billing/", patterns)
        self.assertIn("/tenant/%s/billing/orders/%d/", patterns)
        self.assertIn("/tenant/%s/workspace/%s/task-detail/%s/", patterns)
        self.assertIn("/tenant/%s/people/manage/", patterns)
        self.assertNotIn("/tenant/%s/people/invite/", patterns)
        self.assertIn("/profile/", patterns)


class CheckAllTests(unittest.TestCase):
    def test_current_backend_urls_all_valid(self) -> None:
        # 集成：现网后端产出的全部 action_url 模式必须命中已注册路由或别名。
        self.assertEqual(check_all(), [])

    def test_legacy_redirect_alias_is_legal(self) -> None:
        # 别名 redirect 算合法：老路径 `/billing/gitlab-resources/` 被别名兜底。
        from check_account_deletion_action_urls import _all_route_regexes

        route_res = _all_route_regexes()
        self.assertTrue(any(rx.fullmatch("/tenant/t/billing/gitlab-resources/") for rx in route_res))
        self.assertTrue(any(rx.fullmatch("/tenant/t/settings/members/") for rx in route_res))
        self.assertTrue(any(rx.fullmatch("/tenant/t/workspace/w/task/tk/") for rx in route_res))

    def test_unknown_path_is_rejected(self) -> None:
        from check_account_deletion_action_urls import _all_route_regexes

        route_res = _all_route_regexes()
        self.assertFalse(any(rx.fullmatch("/tenant/t/billing/unknown-page/") for rx in route_res))
        self.assertFalse(any(rx.fullmatch("/billing/gitlab-resources/") for rx in route_res))


if __name__ == "__main__":
    unittest.main()
