"""layout-from-config-repo.sh assembles $DEPLOY_ROOT from daydaymoney-deploy without ram-work."""

from __future__ import annotations

import os
import stat
import subprocess
from pathlib import Path

import pytest

LAYOUT = Path(__file__).resolve().parents[1] / "layout-from-config-repo.sh"


def _write(p: Path, text: str = "ok\n") -> None:
    p.parent.mkdir(parents=True, exist_ok=True)
    p.write_text(text, encoding="utf-8")
    if p.suffix == ".sh":
        p.chmod(0o755)


def _seed_config_repo(repo: Path) -> None:
    env = repo / "envs" / "current"
    _write(env / "conf" / "base.yaml", "scheme: https\n")
    _write(env / "conf" / "runAll.yaml", "groups: []\n")
    _write(env / "releases.yaml", "artifacts: {}\n")
    _write(env / "dockerInfra" / "mysql" / "run.sh")
    _write(env / "gitService" / "run.sh")
    _write(env / "AiMonitor" / "run.sh")
    _write(env / "taskGateway" / "run.sh")
    _write(env / "taskSSE" / "run.sh")
    _write(env / "taskEvents" / "run.sh")
    _write(env / "taskFE" / "docker-compose.yml", "services: {}\n")
    _write(env / "taskFE" / "app" / "scripts" / "runall-lifecycle.sh")
    _write(env / "dataMigrate" / "taskAuth" / "001.sql", "SELECT 1;\n")
    _write(env / "db" / "registry.yaml", "version: '2'\n")
    _write(env / "runAll" / "scripts" / "conf-read.py", "print(1)\n")
    _write(env / "trae-agent" / "onlineServiceJS" / "run.sh")


def _run_layout(config_repo: Path, deploy_root: Path) -> subprocess.CompletedProcess[str]:
    return subprocess.run(
        ["bash", str(LAYOUT)],
        env={
            **os.environ,
            "CONFIG_REPO": str(config_repo),
            "DEPLOY_ROOT": str(deploy_root),
            "DEPLOY_ENV": "current",
        },
        capture_output=True,
        text=True,
        check=False,
    )


def test_layout_to_separate_deploy_root(tmp_path):
    repo = tmp_path / "daydaymoney-deploy"
    dest = tmp_path / "ram-deploy"
    _seed_config_repo(repo)
    dest.mkdir()
    r = _run_layout(repo, dest)
    assert r.returncode == 0, r.stderr + r.stdout
    assert (dest / "conf" / "base.yaml").is_file()
    assert (dest / "dockerInfra" / "mysql" / "run.sh").is_file()
    assert (dest / "dataMigrate" / "taskAuth" / "001.sql").is_file()
    assert (dest / "db" / "registry.yaml").is_file()
    assert (dest / "runAll" / "scripts" / "conf-read.py").is_file()
    gw_logs = dest / "taskGateway" / "logs"
    assert gw_logs.is_dir()
    mode = stat.S_IMODE(gw_logs.stat().st_mode)
    assert mode == 0o777
    assert not (dest / "dockerInfra" / "mysql" / "data").exists()


def test_layout_does_not_overwrite_host_releases(tmp_path):
    repo = tmp_path / "daydaymoney-deploy"
    dest = tmp_path / "ram-deploy"
    _seed_config_repo(repo)
    dest.mkdir()
    _write(dest / "releases.yaml", "artifacts:\n  runAll: {sha: host, package: file:///x}\n")
    r = _run_layout(repo, dest)
    assert r.returncode == 0, r.stderr
    text = (dest / "releases.yaml").read_text(encoding="utf-8")
    assert "file:///x" in text


def test_layout_clone_as_root_uses_symlinks(tmp_path):
    repo = tmp_path / "daydaymoney-deploy"
    _seed_config_repo(repo)
    r = _run_layout(repo, repo)
    assert r.returncode == 0, r.stderr + r.stdout
    conf = repo / "conf"
    assert conf.is_symlink()
    assert conf.resolve() == (repo / "envs" / "current" / "conf").resolve()
    assert (repo / "conf" / "base.yaml").is_file()


def test_layout_clone_as_root_rewrites_file_root_via_conf_local(tmp_path):
    repo = tmp_path / "daydaymoney-deploy"
    _seed_config_repo(repo)
    tracked = repo / "envs" / "current" / "conf" / "runAll.yaml"
    tracked.write_text("logging:\n  file_root: /tmp/ram-work/logs\ngroups: []\n", encoding="utf-8")
    overlay = repo / "conf-local" / "runAll.yaml"
    overlay.parent.mkdir(parents=True)
    overlay.write_text(
        "groups:\n  - services:\n      - env:\n          SECRET: keep-me\n",
        encoding="utf-8",
    )
    r = _run_layout(repo, repo)
    assert r.returncode == 0, r.stderr + r.stdout
    assert "file_root: /tmp/ram-work/logs" in tracked.read_text(encoding="utf-8")
    body = overlay.read_text(encoding="utf-8")
    assert "keep-me" in body
    assert f"file_root: {repo / 'logs'}" in body or str(repo / "logs") in body


def test_layout_separate_root_rewrites_copied_runall_file_root(tmp_path):
    repo = tmp_path / "daydaymoney-deploy"
    dest = tmp_path / "ram-deploy"
    _seed_config_repo(repo)
    (repo / "envs" / "current" / "conf" / "runAll.yaml").write_text(
        "logging:\n  file_root: /tmp/ram-work/logs\ngroups: []\n",
        encoding="utf-8",
    )
    dest.mkdir()
    r = _run_layout(repo, dest)
    assert r.returncode == 0, r.stderr + r.stdout
    copied = (dest / "conf" / "runAll.yaml").read_text(encoding="utf-8")
    assert f"file_root: {dest}/logs" in copied
    overlay = (dest / "conf-local" / "runAll.yaml").read_text(encoding="utf-8")
    assert str(dest / "logs") in overlay


def test_layout_removes_ram_work_mysql_data_symlink(tmp_path):
    ram = Path("/tmp/ram-work/dockerInfra/mysql/data")
    if not (ram.exists() or ram.is_symlink()):
        pytest.skip("no /tmp/ram-work mysql data to simulate P4 cutover symlink")
    repo = tmp_path / "daydaymoney-deploy"
    dest = tmp_path / "ram-deploy"
    _seed_config_repo(repo)
    dest.mkdir()
    data = dest / "dockerInfra" / "mysql" / "data"
    data.parent.mkdir(parents=True)
    os.symlink(ram, data)
    r = _run_layout(repo, dest)
    assert r.returncode == 0, r.stderr + r.stdout
    assert not data.is_symlink()
    assert not data.exists()
