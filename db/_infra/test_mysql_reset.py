"""Regression tests for mysql_reset.py.

The 9999 「清空全部数据库」 button used to call mysql-reset.sh which only
DROP+CREATE databases listed in db/registry.yaml. That left:

1. leftover test databases (OpenTestMySQL / infra tests) and their datadir
   directories under dockerInfra/mysql/data/
2. binary logs (default expire 30 days) which dominate datadir size

These tests pin the replacement policy: drop every non-system schema, recreate
only registry databases, and purge binary logs. No live MySQL is required.
"""

from __future__ import annotations

import unittest

import mysql_reset as mr

EXISTING_WITH_LEFTOVERS = [
    "ai_provider",
    "container",
    "git_oauth",
    "information_schema",
    "mysql",
    "performance_schema",
    "sys",
    "task_auth",
    "task_auth_test_e0e9rdg9",
    "task_cloud",
    "task_cloud_test_4zsjtgrm",
    "task_task_test_t0xkqxod",
    "test_infra_testscantokennullexpires",
    "test_task_tenant",
]

REGISTRY = {
    "mysql": {"host": "127.0.0.1"},
    "databases": {
        "task-auth": {"database": "task_auth"},
        "task-cloud": {"database": "task_cloud"},
        "container": {"database": "container"},
        "git-oauth": {"database": "git_oauth"},
        "ai-provider": {"database": "ai_provider"},
    },
}


class TestUserDatabasesToDrop(unittest.TestCase):
    def test_excludes_system_schemas(self):
        dropped = mr.user_databases_to_drop(EXISTING_WITH_LEFTOVERS)
        for sysdb in ("mysql", "information_schema", "performance_schema", "sys"):
            self.assertNotIn(sysdb, dropped)

    def test_includes_leftover_test_databases(self):
        """The bug: registry-only DROP left these directories unchanged."""
        dropped = mr.user_databases_to_drop(EXISTING_WITH_LEFTOVERS)
        for leftover in (
            "task_auth_test_e0e9rdg9",
            "task_cloud_test_4zsjtgrm",
            "task_task_test_t0xkqxod",
            "test_infra_testscantokennullexpires",
            "test_task_tenant",
        ):
            self.assertIn(leftover, dropped)

    def test_includes_registry_databases_so_they_are_wiped(self):
        dropped = mr.user_databases_to_drop(EXISTING_WITH_LEFTOVERS)
        for name in ("task_auth", "task_cloud", "container", "git_oauth", "ai_provider"):
            self.assertIn(name, dropped)

    def test_registry_only_policy_would_miss_leftovers(self):
        registry_names = set(mr.registry_database_names(REGISTRY))
        missed = [
            name
            for name in EXISTING_WITH_LEFTOVERS
            if name not in registry_names and name not in mr.SYSTEM_DATABASES
        ]
        self.assertEqual(
            missed,
            [
                "task_auth_test_e0e9rdg9",
                "task_cloud_test_4zsjtgrm",
                "task_task_test_t0xkqxod",
                "test_infra_testscantokennullexpires",
                "test_task_tenant",
            ],
        )
        dropped = mr.user_databases_to_drop(EXISTING_WITH_LEFTOVERS)
        for name in missed:
            self.assertIn(name, dropped)


class TestRegistryDatabaseNames(unittest.TestCase):
    def test_reads_database_field_not_yaml_key(self):
        names = mr.registry_database_names(REGISTRY)
        self.assertEqual(
            names,
            ["ai_provider", "container", "git_oauth", "task_auth", "task_cloud"],
        )

    def test_falls_back_to_entry_key_when_database_missing(self):
        names = mr.registry_database_names(
            {"databases": {"task-bill": {}, "task-auth": {"database": "task_auth"}}}
        )
        self.assertEqual(names, ["task-bill", "task_auth"])


class TestBuildResetSql(unittest.TestCase):
    def setUp(self):
        self.registry_dbs = mr.registry_database_names(REGISTRY)
        self.sql = mr.build_reset_sql(EXISTING_WITH_LEFTOVERS, self.registry_dbs)

    def test_drops_leftover_test_schemas(self):
        self.assertIn("DROP DATABASE IF EXISTS `task_cloud_test_4zsjtgrm`;", self.sql)
        self.assertIn("DROP DATABASE IF EXISTS `test_task_tenant`;", self.sql)
        self.assertIn("DROP DATABASE IF EXISTS `test_infra_testscantokennullexpires`;", self.sql)

    def test_never_drops_system_schemas(self):
        for sysdb in ("mysql", "information_schema", "performance_schema", "sys"):
            self.assertNotIn(f"DROP DATABASE IF EXISTS `{sysdb}`;", self.sql)

    def test_recreates_only_registry_databases(self):
        self.assertIn(
            "CREATE DATABASE `task_auth` CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;",
            self.sql,
        )
        self.assertNotIn("CREATE DATABASE `task_cloud_test_4zsjtgrm`", self.sql)
        self.assertNotIn("CREATE DATABASE `test_task_tenant`", self.sql)

    def test_disables_binlog_before_ddl_then_purges_logs(self):
        disable_at = self.sql.find("SET SESSION sql_log_bin = 0;")
        first_drop = self.sql.find("DROP DATABASE IF EXISTS")
        first_reset = self.sql.find("RESET MASTER;")
        last_reset = self.sql.rfind("RESET MASTER;")
        self.assertNotEqual(disable_at, -1)
        self.assertLess(disable_at, first_drop)
        self.assertLess(first_reset, first_drop)
        self.assertGreater(last_reset, first_drop)
        self.assertGreaterEqual(self.sql.count("RESET MASTER;"), 2)

    def test_rejects_unsafe_database_names(self):
        with self.assertRaises(ValueError):
            mr.build_reset_sql(["task_auth`; DROP DATABASE mysql; --"], ["task_auth"])


class TestParseKillableSessionIds(unittest.TestCase):
    def test_skips_header_and_parses_ids(self):
        raw = "ID\n113208\n113209\n"
        self.assertEqual(mr.parse_killable_session_ids(raw), [113208, 113209])

    def test_ignores_blank_lines(self):
        self.assertEqual(mr.parse_killable_session_ids("\n42\n\n"), [42])

    def test_skips_mysql_password_warning_lines(self):
        raw = "mysql: [Warning] Using a password on the command line interface can be insecure.\n113208\n"
        self.assertEqual(mr.parse_killable_session_ids(raw), [113208])


if __name__ == "__main__":
    unittest.main()

