"""Promtail scrape root prefers live P4 logs over ram-work/logs (OPT-20260831-004)."""
from __future__ import annotations

import os
import subprocess
import tempfile
import unittest
from pathlib import Path

SCRIPTS_DIR = Path(__file__).resolve().parent.parent
SCRIPT = SCRIPTS_DIR / "runall-local-promtail.sh"


def _print_log_root(extra: dict[str, str]) -> str:
    merged = os.environ.copy()
    merged.pop("RUNALL_LOG_ROOT", None)
    merged.update(extra)
    proc = subprocess.run(
        ["bash", str(SCRIPT), "print-log-root"],
        capture_output=True,
        text=True,
        env=merged,
        check=False,
    )
    if proc.returncode != 0:
        raise AssertionError(proc.stderr or proc.stdout)
    return proc.stdout.strip()


class TestResolveRunallLogRoot(unittest.TestCase):
    def test_explicit_env_wins(self):
        got = _print_log_root({"RUNALL_LOG_ROOT": "/tmp/explicit-runall-logs"})
        self.assertEqual(got, "/tmp/explicit-runall-logs")

    def test_prefers_deploy_root_when_task_log_exists(self):
        with tempfile.TemporaryDirectory() as raw:
            deploy = Path(raw)
            logs = deploy / "logs"
            logs.mkdir()
            (logs / "task-task-service.log").write_text("x\n")
            got = _print_log_root({"DEPLOY_ROOT": str(deploy)})
            self.assertEqual(got, str(logs))

    def test_script_discovers_live_runall_environ(self):
        text = SCRIPT.read_text(encoding="utf-8")
        self.assertIn("/proc/", text)
        self.assertIn("RUNALL_LOG_ROOT", text)
        self.assertIn("pgrep", text)


if __name__ == "__main__":
    unittest.main(verbosity=2)
