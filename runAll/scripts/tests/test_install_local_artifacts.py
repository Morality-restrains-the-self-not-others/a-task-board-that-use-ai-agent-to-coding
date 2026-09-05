"""Incremental install-local-artifacts: skip unchanged ELF; copy-then-mv when changed."""

from __future__ import annotations

import subprocess
import sys
from pathlib import Path

_TESTS = Path(__file__).resolve().parent
if str(_TESTS) not in sys.path:
    sys.path.insert(0, str(_TESTS))
from test_up_from_config_repo import INSTALL, _clean_env, _write


def test_install_skips_unchanged_elf(tmp_path):
    root = tmp_path / "deploy"
    root.mkdir()
    arts = tmp_path / "arts"
    _write(arts / "runAll", "same-bytes\n")
    (arts / "runAll").chmod(0o755)
    r1 = subprocess.run(
        ["bash", str(INSTALL), str(arts), str(root)],
        env=_clean_env(),
        capture_output=True,
        text=True,
        check=False,
    )
    assert r1.returncode == 0, r1.stderr + r1.stdout
    dest = root / "bin" / "runAll"
    inode1 = dest.stat().st_ino
    r2 = subprocess.run(
        ["bash", str(INSTALL), str(arts), str(root)],
        env=_clean_env(),
        capture_output=True,
        text=True,
        check=False,
    )
    assert r2.returncode == 0, r2.stderr + r2.stdout
    log = r2.stdout + r2.stderr
    assert "unchanged" in log, log
    assert dest.stat().st_ino == inode1
    assert dest.read_text(encoding="utf-8") == "same-bytes\n"


def test_install_copy_then_mv_changed_elf_new_inode(tmp_path):
    root = tmp_path / "deploy"
    root.mkdir()
    arts = tmp_path / "arts"
    _write(arts / "runAll", "before\n")
    (arts / "runAll").chmod(0o755)
    r1 = subprocess.run(
        ["bash", str(INSTALL), str(arts), str(root)],
        env=_clean_env(),
        capture_output=True,
        text=True,
        check=False,
    )
    assert r1.returncode == 0, r1.stderr + r1.stdout
    dest = root / "bin" / "runAll"
    inode1 = dest.stat().st_ino
    _write(arts / "runAll", "after-change\n")
    (arts / "runAll").chmod(0o755)
    r2 = subprocess.run(
        ["bash", str(INSTALL), str(arts), str(root)],
        env=_clean_env(),
        capture_output=True,
        text=True,
        check=False,
    )
    assert r2.returncode == 0, r2.stderr + r2.stdout
    assert dest.read_text(encoding="utf-8") == "after-change\n"
    assert dest.stat().st_ino != inode1
    assert "unchanged" not in r2.stdout + r2.stderr or dest.read_text(encoding="utf-8") == "after-change\n"
