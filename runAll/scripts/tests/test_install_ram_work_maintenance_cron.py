#!/usr/bin/env python3
"""Nightly SonarQube cron must be installed at 03:00 via the maintenance installer."""

from __future__ import annotations

import subprocess
import unittest
from pathlib import Path

SCRIPTS = Path(__file__).resolve().parents[1]
ROOT = SCRIPTS.parents[1]
INSTALLER = SCRIPTS / "install-ram-work-maintenance-cron.sh"
SONAR_SH = ROOT / "sonarqube.sh"


class InstallRamWorkMaintenanceCronTest(unittest.TestCase):
    def test_dry_run_schedules_sonarqube_at_0300_nightly(self) -> None:
        out = subprocess.check_output(
            ["bash", str(INSTALLER), "--dry-run"],
            text=True,
        )
        self.assertIn("0 3 * * *", out)
        self.assertIn("sonarqube.sh", out)
        sonar_lines = [ln for ln in out.splitlines() if "sonarqube.sh" in ln]
        self.assertEqual(len(sonar_lines), 1, out)
        self.assertTrue(
            sonar_lines[0].startswith("0 3 * * *"),
            sonar_lines[0],
        )

    def test_sonarqube_sh_uses_flock_to_avoid_overlapping_scans(self) -> None:
        text = SONAR_SH.read_text(encoding="utf-8")
        self.assertIn("flock", text)
        self.assertIn("sonarqube-scan.lock", text)

    def test_dry_run_keeps_existing_maintenance_jobs(self) -> None:
        out = subprocess.check_output(
            ["bash", str(INSTALLER), "--dry-run"],
            text=True,
        )
        self.assertIn("truncate-ram-work-logs.sh", out)
        self.assertIn("prune-legacy-binaries", out)


if __name__ == "__main__":
    unittest.main()
