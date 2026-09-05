#!/usr/bin/env python3
"""Regression for negative_id_migration_plan.py — inventory + no PK UPDATE."""
from __future__ import annotations

import subprocess
import sys
import unittest
from pathlib import Path

SCRIPT = Path(__file__).resolve().parent / "negative_id_migration_plan.py"


class NegativeIDMigrationPlanTest(unittest.TestCase):
    def test_check_prints_inventory_and_expand_without_pk_update(self) -> None:
        proc = subprocess.run(
            [sys.executable, str(SCRIPT), "--check"],
            check=False,
            capture_output=True,
            text=True,
        )
        self.assertEqual(proc.returncode, 0, proc.stderr)
        out = proc.stdout
        self.assertIn("INVENTORY workspaces=2 projects=1", out)
        self.assertIn("ws_-2309487803472456748", out)
        self.assertIn("ws_-2740859684112864748", out)
        self.assertIn("proj_-2304947540687519745", out)
        self.assertIn("CREATE TABLE IF NOT EXISTS project_id_alias", out)
        self.assertIn("utf8mb4", out)
        self.assertNotIn("UPDATE project_workspace_entries SET id=", out)
        self.assertNotIn("UPDATE project_entries SET id=", out)
        self.assertIn("Forbidden: lock-table UPDATE of the primary key", out)

    def test_apply_is_rejected(self) -> None:
        proc = subprocess.run(
            [sys.executable, str(SCRIPT), "--apply"],
            check=False,
            capture_output=True,
            text=True,
        )
        self.assertEqual(proc.returncode, 2)
        self.assertIn("refuses live PK UPDATE", proc.stderr)


if __name__ == "__main__":
    unittest.main()
