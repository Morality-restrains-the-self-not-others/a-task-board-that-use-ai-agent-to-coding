#!/usr/bin/env python3
"""Regression: local docker-mysql durability flags stay at test-friendly values."""
from __future__ import annotations

import tempfile
import unittest
from pathlib import Path

from load_mysql_conf import export_lines, load_mysql_conf

ROOT = Path(__file__).resolve().parents[2]
CONF = ROOT / "conf" / "infra" / "mysql" / "config.yaml"
COMPOSE = Path(__file__).resolve().parent / "docker-compose.yml"


class TestMysqlDurabilityConf(unittest.TestCase):
    def test_ssot_conf_relaxes_fsync_for_local_instance(self) -> None:
        conf = load_mysql_conf(str(CONF))
        self.assertEqual(conf["innodbFlushLogAtTrxCommit"], 2)
        self.assertEqual(conf["syncBinlog"], 0)
        self.assertTrue(conf["skipLogBin"])

    def test_export_matches_compose_env_names(self) -> None:
        lines = export_lines(load_mysql_conf(str(CONF)))
        joined = "\n".join(lines)
        self.assertIn("MYSQL_INNODB_FLUSH_LOG_AT_TRX_COMMIT=2", joined)
        self.assertIn("MYSQL_SYNC_BINLOG=0", joined)
        self.assertIn("MYSQL_SKIP_LOG_BIN=1", joined)

    def test_compose_consumes_flush_and_skips_binlog(self) -> None:
        text = COMPOSE.read_text(encoding="utf-8")
        self.assertIn("MYSQL_INNODB_FLUSH_LOG_AT_TRX_COMMIT", text)
        self.assertIn("MYSQL_SYNC_BINLOG", text)
        self.assertIn("--skip-log-bin", text)
        self.assertNotIn("binlog_expire_logs_seconds", text)

    def test_override_file_changes_export(self) -> None:
        with tempfile.NamedTemporaryFile("w", suffix=".yaml", delete=False) as fh:
            fh.write("innodbFlushLogAtTrxCommit: 1\nsyncBinlog: 1\nskipLogBin: false\n")
            path = fh.name
        conf = load_mysql_conf(path)
        self.assertEqual(conf["innodbFlushLogAtTrxCommit"], 1)
        self.assertEqual(conf["syncBinlog"], 1)
        self.assertFalse(conf["skipLogBin"])
        joined = "\n".join(export_lines(conf))
        self.assertIn("MYSQL_SKIP_LOG_BIN=0", joined)


if __name__ == "__main__":
    unittest.main()
