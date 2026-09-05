"""conf-sync 须扫描一级 conf/<app>/（如 task-referral），不能只扫 conf/<area>/<app>/。"""

from __future__ import annotations

import importlib.util
import sys
from pathlib import Path

import pytest
import yaml

SCRIPTS = Path(__file__).resolve().parents[1]
CONF_SYNC = SCRIPTS / "conf-sync.py"
CONF_SYNC_ALL = SCRIPTS / "conf-sync-all.sh"


def _load_conf_sync():
    spec = importlib.util.spec_from_file_location("conf_sync_under_test", CONF_SYNC)
    mod = importlib.util.module_from_spec(spec)
    assert spec.loader is not None
    spec.loader.exec_module(mod)
    return mod


@pytest.fixture
def conf_sync():
    return _load_conf_sync()


def test_conf_sync_all_scans_one_level_dirs():
    text = CONF_SYNC_ALL.read_text(encoding="utf-8")
    assert '"$ROOT"/conf/*/' in text
    assert "sync.manifest.yaml" in text


def test_main_syncs_one_level_app(tmp_path, monkeypatch, conf_sync):
    bill = tmp_path / "conf" / "billing" / "task-bill"
    bill.mkdir(parents=True)
    (bill / "config.yaml").write_text("internalSecret: test-bill-secret\n", encoding="utf-8")
    app = tmp_path / "conf" / "task-referral"
    app.mkdir(parents=True)
    (app / "sync.manifest.yaml").write_text(
        "version: '1'\n"
        "fragments:\n"
        "  - from: ../billing/task-bill/config.yaml\n"
        "    to: task-bill.yaml\n"
        "    pick: [internalSecret]\n",
        encoding="utf-8",
    )
    # 两级目录不应挡住一级 app
    nested = tmp_path / "conf" / "auth" / "task-auth"
    nested.mkdir(parents=True)

    monkeypatch.setattr(conf_sync, "repo_root", lambda: tmp_path)
    sys.path.insert(0, str(SCRIPTS))
    import conf_lib

    monkeypatch.setattr(conf_lib, "repo_root", lambda: tmp_path)

    conf_sync.main([])
    out = app / "task-bill.yaml"
    assert out.is_file(), "一级 conf/task-referral 必须被 conf-sync 全量循环同步"
    data = yaml.safe_load(out.read_text(encoding="utf-8"))
    assert data["internalSecret"] == "test-bill-secret"
