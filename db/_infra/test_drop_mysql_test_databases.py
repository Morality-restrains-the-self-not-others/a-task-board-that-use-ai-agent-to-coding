"""Unit tests for drop_mysql_test_databases.py (no live MySQL required)."""

from __future__ import annotations

import unittest

import drop_mysql_test_databases as d
import mysql_reset as mr

EXISTING = [
    "information_schema",
    "mysql",
    "performance_schema",
    "sys",
    "task_auth",
    "task_bill",
    "task_auth_test_e0e9rdg9",
    "task_cloud_test_4zsjtgrm",
    "dbload_clone_ut_test_zeh5vj70",
    "test_infra_testscantokennullexpires",
    "test_task_tenant",
    "tpl_task_auth_962d19bde286",
    "tpl_dbload_clone_ut_clone-ut-wid",
]

REGISTRY = [
    "task_auth",
    "task_bill",
    "task_cloud",
]


class TestClassifyLeftovers(unittest.TestCase):
    def test_drops_opentestmysql_names_not_registry(self):
        got = d.classify_leftovers(EXISTING, REGISTRY, include_templates=False)
        self.assertIn("task_auth_test_e0e9rdg9", got)
        self.assertIn("task_cloud_test_4zsjtgrm", got)
        self.assertIn("dbload_clone_ut_test_zeh5vj70", got)
        self.assertIn("test_infra_testscantokennullexpires", got)
        self.assertIn("test_task_tenant", got)
        self.assertNotIn("task_auth", got)
        self.assertNotIn("task_bill", got)
        self.assertNotIn("mysql", got)
        self.assertNotIn("tpl_task_auth_962d19bde286", got)

    def test_never_drops_system_or_registry_even_if_name_looks_like_test(self):
        self.assertFalse(d.is_leftover_test_schema("mysql", REGISTRY))
        self.assertFalse(d.is_leftover_test_schema("task_auth", REGISTRY))
        self.assertFalse(d.is_leftover_test_schema("information_schema", []))

    def test_include_templates_adds_tpl_only(self):
        got = d.classify_leftovers(EXISTING, REGISTRY, include_templates=True)
        self.assertIn("tpl_task_auth_962d19bde286", got)
        self.assertIn("task_auth_test_e0e9rdg9", got)
        self.assertNotIn("task_auth", got)

    def test_build_drop_sql_only_drops_named_schemas(self):
        sql = d.build_drop_sql(["task_auth_test_e0e9rdg9", "tpl_task_auth_abc"])
        self.assertIn("DROP DATABASE IF EXISTS `task_auth_test_e0e9rdg9`;", sql)
        self.assertNotIn("DROP DATABASE IF EXISTS `task_auth`;", sql)
        self.assertNotIn("DROP DATABASE IF EXISTS `mysql`;", sql)

    def test_rejects_injection_in_db_name(self):
        with self.assertRaises(ValueError):
            mr.validate_db_name("task_auth`; DROP DATABASE mysql; --")


if __name__ == "__main__":
    unittest.main()
