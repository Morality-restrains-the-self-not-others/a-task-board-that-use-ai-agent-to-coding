"""check_p4_deploy_root.sh rejects source trees (ADR-0052 P4)."""

from __future__ import annotations

import os
import subprocess
from pathlib import Path

import pytest

SCRIPT = Path(__file__).resolve().parents[1] / "check_p4_deploy_root.sh"


def _make_min_deploy(root: Path) -> None:
    (root / "conf").mkdir(parents=True)
    (root / "conf" / "base.yaml").write_text("scheme: https\n", encoding="utf-8")
    (root / "bin").mkdir()
    for name in ("runAll", "taskAuth"):
        p = root / "bin" / name
        p.write_text("#!/bin/sh\nexit 0\n", encoding="utf-8")
        p.chmod(0o755)
    js = root / "trae-agent" / "onlineServiceJS"
    (js / "src").mkdir(parents=True)
    (js / "run.sh").write_text("#!/bin/sh\nexit 0\n", encoding="utf-8")
    (js / "run.sh").chmod(0o755)
    (js / "Dockerfile").write_text("FROM scratch\n", encoding="utf-8")
    (js / "buildDocker.sh").write_text("#!/bin/sh\nexit 0\n", encoding="utf-8")
    (js / "src" / "server.mjs").write_text("export {}\n", encoding="utf-8")


def _run(root: Path) -> subprocess.CompletedProcess[str]:
    return subprocess.run(
        ["bash", str(SCRIPT), str(root)],
        capture_output=True,
        text=True,
        check=False,
    )


def test_ok_minimal_deploy_root(tmp_path):
    _make_min_deploy(tmp_path)
    r = _run(tmp_path)
    assert r.returncode == 0, r.stderr
    assert "ok" in r.stdout


def test_fails_on_go_source(tmp_path):
    _make_min_deploy(tmp_path)
    (tmp_path / "taskAuth").mkdir()
    (tmp_path / "taskAuth" / "main.go").write_text("package main\n", encoding="utf-8")
    r = _run(tmp_path)
    assert r.returncode != 0
    assert "Go source" in r.stderr


def test_fails_on_gitmodules(tmp_path):
    _make_min_deploy(tmp_path)
    (tmp_path / ".gitmodules").write_text("[submodule \"x\"]\n", encoding="utf-8")
    r = _run(tmp_path)
    assert r.returncode != 0
    assert ".gitmodules" in r.stderr


def test_fails_on_ram_work_symlink(tmp_path):
    _make_min_deploy(tmp_path)
    src = Path("/tmp/ram-work")
    if not src.is_dir():
        pytest.skip("no /tmp/ram-work on this host")
    link = tmp_path / "taskAuth"
    os.symlink(src / "taskAuth", link)
    r = _run(tmp_path)
    assert r.returncode != 0
    assert "source symlink" in r.stderr


def test_allows_git_dir_without_scanning_go_inside(tmp_path):
    _make_min_deploy(tmp_path)
    git_obj = tmp_path / ".git" / "objects"
    git_obj.mkdir(parents=True)
    (git_obj / "fake.go").write_text("package main\n", encoding="utf-8")
    r = _run(tmp_path)
    assert r.returncode == 0, r.stderr


def test_allows_sharelib_generated_go(tmp_path):
    _make_min_deploy(tmp_path)
    gen = tmp_path / "shareLib" / "gatewaycors"
    gen.mkdir(parents=True)
    (gen / "allow_headers_gen.go").write_text("package gatewaycors\n", encoding="utf-8")
    r = _run(tmp_path)
    assert r.returncode == 0, r.stderr


def test_fails_mysql_data_corrupt_symlink_to_ram_work(tmp_path):
    _make_min_deploy(tmp_path)
    src = Path("/tmp/ram-work/dockerInfra/mysql/data")
    if not src.exists():
        pytest.skip("no mysql data dir")
    dest = tmp_path / "dockerInfra" / "mysql"
    dest.mkdir(parents=True)
    os.symlink(src, dest / "data.corrupt.20260830_214230")
    r = _run(tmp_path)
    assert r.returncode != 0
    assert "source symlink" in r.stderr


def test_fails_mysql_data_symlink_to_ram_work(tmp_path):
    _make_min_deploy(tmp_path)
    src = Path("/tmp/ram-work/dockerInfra/mysql/data")
    if not src.exists():
        pytest.skip("no mysql data dir")
    dest = tmp_path / "dockerInfra" / "mysql"
    dest.mkdir(parents=True)
    os.symlink(src, dest / "data")
    r = _run(tmp_path)
    assert r.returncode != 0
    assert "source symlink" in r.stderr
