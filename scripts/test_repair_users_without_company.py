#!/usr/bin/env python3
"""Unit tests for repair_users_without_company skip logic.

受邀加入他人公司的用户不得被跳过；仅「作为创建者已有公司」才 skip。
"""

from __future__ import annotations

import importlib.util
import unittest
from pathlib import Path
from unittest import mock


def _load_module():
    path = Path(__file__).resolve().parent / "repair_users_without_company.py"
    spec = importlib.util.spec_from_file_location("repair_users_without_company", path)
    mod = importlib.util.module_from_spec(spec)
    assert spec.loader is not None
    spec.loader.exec_module(mod)
    return mod


class TestUserHasOwnCompany(unittest.TestCase):
    def setUp(self):
        self.mod = _load_module()

    def test_invited_member_only_is_not_own_company(self):
        """members/exists=true 但 by-creator=[] → 仍需重放建自有公司。"""
        with mock.patch.object(self.mod, "_http_get") as get:
            get.return_value = (200, [])
            self.assertFalse(self.mod.user_has_own_company("874599872605483008"))
            url = get.call_args[0][0]
            self.assertIn("companies/by-creator", url)
            self.assertIn("creator_id=874599872605483008", url)

    def test_creator_has_own_company(self):
        with mock.patch.object(self.mod, "_http_get") as get:
            get.return_value = (
                200,
                [{"id": "874599492341493760", "name": "team1", "creator_id": "u1"}],
            )
            self.assertTrue(self.mod.user_has_own_company("u1"))

    def test_fire_skips_only_when_own_company(self):
        with mock.patch.object(self.mod, "user_has_own_company", return_value=True):
            result = self.mod.fire_user_created("u1", dry_run=True)
        self.assertEqual(result["action"], "skip")
        self.assertIn("by-creator", result["detail"])

    def test_fire_would_replay_for_invited_only_user(self):
        with mock.patch.object(self.mod, "user_has_own_company", return_value=False), mock.patch.object(
            self.mod, "resolve_user_info", return_value=("", "")
        ):
            result = self.mod.fire_user_created("874599872605483008", dry_run=True)
        self.assertEqual(result["action"], "would_fire")
        self.assertEqual(result["username"], "我的公司")


if __name__ == "__main__":
    unittest.main()
