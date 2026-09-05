"""conf_lib.repo_root honors CONF_ROOT / DEPLOY_ROOT (ADR-0052)."""

from __future__ import annotations

import importlib
import sys
from pathlib import Path

import pytest

SCRIPTS = Path(__file__).resolve().parents[1]
if str(SCRIPTS) not in sys.path:
    sys.path.insert(0, str(SCRIPTS))


@pytest.fixture
def conf_lib(monkeypatch):
    monkeypatch.delenv("CONF_ROOT", raising=False)
    monkeypatch.delenv("DEPLOY_ROOT", raising=False)
    import conf_lib

    importlib.reload(conf_lib)
    return conf_lib


def _write_base(conf_dir: Path) -> None:
    conf_dir.mkdir(parents=True, exist_ok=True)
    (conf_dir / "base.yaml").write_text("scheme: https\n", encoding="utf-8")


def test_repo_root_from_conf_root_conf_dir(tmp_path, monkeypatch, conf_lib):
    deploy = tmp_path / "deploy"
    conf_dir = deploy / "conf"
    _write_base(conf_dir)
    unrelated = tmp_path / "unrelated"
    unrelated.mkdir()
    monkeypatch.chdir(unrelated)
    monkeypatch.setenv("CONF_ROOT", str(conf_dir))
    monkeypatch.delenv("DEPLOY_ROOT", raising=False)

    assert conf_lib.repo_root() == deploy.resolve()


def test_repo_root_from_deploy_root(tmp_path, monkeypatch, conf_lib):
    deploy = tmp_path / "deploy"
    _write_base(deploy / "conf")
    unrelated = tmp_path / "unrelated"
    unrelated.mkdir()
    monkeypatch.chdir(unrelated)
    monkeypatch.delenv("CONF_ROOT", raising=False)
    monkeypatch.setenv("DEPLOY_ROOT", str(deploy))

    assert conf_lib.repo_root() == deploy.resolve()
