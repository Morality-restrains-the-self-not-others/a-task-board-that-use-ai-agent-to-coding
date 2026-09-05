#!/usr/bin/env python3
"""Tests for disk_capacity_reporter.py (OPT-20260823-066).

Run: python3 scripts/test_disk_capacity_reporter.py
"""

from __future__ import annotations

import os
import subprocess
import sys
import tempfile
import unittest
from pathlib import Path

from disk_capacity_reporter import collect_metrics, dir_size_bytes


class DirSizeTests(unittest.TestCase):
    def test_empty_dir_is_zero(self):
        with tempfile.TemporaryDirectory() as d:
            self.assertEqual(dir_size_bytes(d), 0)

    def test_sums_nested_files(self):
        with tempfile.TemporaryDirectory() as d:
            (Path(d) / "a.pdf").write_bytes(b"x" * 100)
            sub = Path(d) / "sub"
            sub.mkdir()
            (sub / "b.pdf").write_bytes(b"y" * 250)
            (sub / "note.txt").write_bytes(b"z" * 50)
            self.assertEqual(dir_size_bytes(d), 400)

    def test_missing_dir_is_zero(self):
        self.assertEqual(dir_size_bytes("/nonexistent-opt-066-dir"), 0)


class CollectMetricsTests(unittest.TestCase):
    def test_emits_expected_metric_names(self):
        with tempfile.TemporaryDirectory() as d:
            (Path(d) / "inv.pdf").write_bytes(b"a" * 10)
            lines = collect_metrics(paths=[d], data_dirs=[d])
            metric_lines = [l for l in lines if l and not l.startswith("#")]
            names = {l.split("{")[0] for l in metric_lines}
            self.assertIn("repo_tmpfs_size_bytes", names)
            self.assertIn("repo_tmpfs_used_bytes", names)
            self.assertIn("repo_tmpfs_avail_bytes", names)
            self.assertIn("repo_data_dir_size_bytes", names)

    def test_textfile_parseable_format(self):
        with tempfile.TemporaryDirectory() as d:
            lines = collect_metrics(paths=[d], data_dirs=[d])
            for line in lines:
                if line.startswith("#") or not line.strip():
                    continue
                name, _, rest = line.partition("{")
                self.assertTrue(name.startswith("repo_"), f"unexpected metric {line}")
                self.assertIn("} ", rest, f"malformed metric {line}")
                value = rest.rsplit(" ", 1)[-1]
                int(value)  # must be a bare integer

    def test_data_dir_size_reflects_files(self):
        with tempfile.TemporaryDirectory() as d:
            (Path(d) / "f.pdf").write_bytes(b"q" * 7)
            lines = collect_metrics(paths=[d], data_dirs=[d])
            size_line = next(
                l for l in lines
                if l.startswith(f'repo_data_dir_size_bytes{{path="{d}"}}')
            )
            self.assertEqual(size_line.rsplit(" ", 1)[-1], "7")


class CliTests(unittest.TestCase):
    def test_self_test_exits_zero(self):
        proc = subprocess.run(
            [sys.executable, "disk_capacity_reporter.py", "--self-test"],
            capture_output=True, text=True, cwd=os.path.dirname(os.path.abspath(__file__)),
        )
        self.assertEqual(proc.returncode, 0, proc.stderr)
        self.assertIn("SELF-TEST OK", proc.stdout + proc.stderr)

    def test_output_writes_atomic_file(self):
        with tempfile.TemporaryDirectory() as d:
            out = str(Path(d) / "disk_capacity.prom")
            proc = subprocess.run(
                [sys.executable, "disk_capacity_reporter.py", "--output", out],
                capture_output=True, text=True, cwd=os.path.dirname(os.path.abspath(__file__)),
            )
            self.assertEqual(proc.returncode, 0, proc.stderr)
            self.assertTrue(os.path.exists(out))
            content = Path(out).read_text()
            self.assertIn("repo_tmpfs_size_bytes", content)
            self.assertIn("repo_data_dir_size_bytes", content)
            # no leftover .tmp
            self.assertFalse(os.path.exists(out + ".tmp"))


if __name__ == "__main__":
    unittest.main(verbosity=2)
