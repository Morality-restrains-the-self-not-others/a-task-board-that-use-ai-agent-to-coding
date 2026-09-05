"""generate-promtail-config.sh must scrape logs/runall.log (OPT-20260820-009).

Status UI structured lines use service name ``runall`` via
appendStructuredLog(\"runall\"). That name is not a group service in
conf/runAll.yaml, so a generator that only enumerates yaml services never
emits job runall — Loki {job=~\"runall.*\"} stays empty even when the tee
file has ui_request JSON.
"""
from __future__ import annotations

import subprocess
import unittest
from pathlib import Path

SCRIPTS_DIR = Path(__file__).resolve().parent
GENERATOR = SCRIPTS_DIR / "generate-promtail-config.sh"


class TestGeneratePromtailIncludesRunall(unittest.TestCase):
    def test_dry_run_includes_runall_job_and_path(self):
        proc = subprocess.run(
            ["bash", str(GENERATOR), "--dry-run"],
            capture_output=True,
            text=True,
            cwd=str(SCRIPTS_DIR),
            check=False,
        )
        self.assertEqual(proc.returncode, 0, proc.stderr)
        out = proc.stdout
        self.assertIn("job: runall", out)
        self.assertIn("__path__: /var/log/runall/runall.log", out)

    def test_dry_run_pipeline_has_timestamp_stage_for_ts(self):
        """OPT-20260831-021: json 抽取 ts 后必须有 timestamp 阶段，重刮不按摄入时刻写。"""
        proc = subprocess.run(
            ["bash", str(GENERATOR), "--dry-run"],
            capture_output=True,
            text=True,
            cwd=str(SCRIPTS_DIR),
            check=False,
        )
        self.assertEqual(proc.returncode, 0, proc.stderr)
        self.assertIn("- timestamp:", proc.stdout)
        self.assertIn("source: ts", proc.stdout)
        self.assertIn("format: RFC3339Nano", proc.stdout)
        # timestamp 阶段必须在 json 阶段之后（服务级 pipeline_stages 头部）。
        first_job = proc.stdout.find("pipeline_stages:")
        tail = proc.stdout[first_job:]
        self.assertLess(tail.find("source: payload"), tail.find("- timestamp:"))


if __name__ == "__main__":
    unittest.main(verbosity=2)
