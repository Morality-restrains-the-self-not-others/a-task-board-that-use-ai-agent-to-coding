"""Tests: _move_opt.py must scan BLOCK_TODO_*.md for global OPT IDs."""
from __future__ import annotations

import unittest
from pathlib import Path
from tempfile import TemporaryDirectory
from textwrap import dedent

import _move_opt


class ScanBlockTodoFilesTest(unittest.TestCase):
    def test_next_id_sees_ids_in_block_todo_files(self):
        with TemporaryDirectory() as tmp:
            root = Path(tmp)
            (root / "OPTIMIZATION_TODOS.md").write_text("# open\n")
            (root / "OPTIMIZATION_TODOS_COMPLETED.md").write_text("# done\n")
            (root / "PRODUCT_DECISIONS.md").write_text("# product\n")
            (root / "archive" / "completed").mkdir(parents=True)
            (root / "BLOCK_TODO_BROWSER.md").write_text(
                dedent(
                    """\
                    # Blocked
                    ### OPT-20260813-099 — parked
                    - **Status**: pending
                    """
                )
            )
            orig = (
                _move_opt.LEARNINGS,
                _move_opt.OPEN,
                _move_opt.ARCH,
                _move_opt.PRODUCT,
                _move_opt.ARCHIVE_DIR,
            )
            try:
                _move_opt.LEARNINGS = root
                _move_opt.OPEN = root / "OPTIMIZATION_TODOS.md"
                _move_opt.ARCH = root / "OPTIMIZATION_TODOS_COMPLETED.md"
                _move_opt.PRODUCT = root / "PRODUCT_DECISIONS.md"
                _move_opt.ARCHIVE_DIR = root / "archive" / "completed"
                ids = _move_opt.scan_all_opt_ids()
                self.assertIn("OPT-20260813-099", ids)
                self.assertEqual(_move_opt.next_available_id("20260813"), "OPT-20260813-100")
            finally:
                (
                    _move_opt.LEARNINGS,
                    _move_opt.OPEN,
                    _move_opt.ARCH,
                    _move_opt.PRODUCT,
                    _move_opt.ARCHIVE_DIR,
                ) = orig


if __name__ == "__main__":
    unittest.main()
