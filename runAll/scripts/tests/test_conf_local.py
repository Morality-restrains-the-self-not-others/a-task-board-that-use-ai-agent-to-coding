"""conf-local overlay merge."""
from __future__ import annotations

import sys
from pathlib import Path

SCRIPTS = Path(__file__).resolve().parents[1]
if str(SCRIPTS) not in sys.path:
    sys.path.insert(0, str(SCRIPTS))

from conf_lib import deep_merge
from conf_local import merge_conf_local


def test_merge_conf_local_overlays_secret_key(tmp_path, monkeypatch):
    root = tmp_path
    (root / "conf" / "billing" / "paypal").mkdir(parents=True)
    (root / "conf-local" / "billing" / "paypal").mkdir(parents=True)
    (root / "conf-local" / "billing" / "paypal" / "config.yaml").write_text(
        "client_secret: from-local\n", encoding="utf-8"
    )
    monkeypatch.chdir(root)
    got = merge_conf_local(root, "billing/paypal/config.yaml", {"mode": "sandbox", "client_secret": ""})
    assert got["mode"] == "sandbox"
    assert got["client_secret"] == "from-local"


def test_merge_conf_local_missing_is_noop(tmp_path):
    got = merge_conf_local(tmp_path, "billing/paypal/config.yaml", {"mode": "sandbox"})
    assert got == {"mode": "sandbox"}


def test_load_app_config_ignores_config_local_yaml(tmp_path, monkeypatch):
    root = tmp_path
    app = root / "conf" / "core" / "sms"
    app.mkdir(parents=True)
    (root / "conf-local" / "core" / "sms").mkdir(parents=True)
    (root / "conf" / "base.yaml").write_text(
        "scheme: https\nbaseDomain: example.com\nsubdomains: {}\n", encoding="utf-8"
    )
    (app / "config.yaml").write_text("sms:\n  aliyun:\n    access_key_id: \"\"\n", encoding="utf-8")
    (root / "conf-local" / "core" / "sms" / "config.yaml").write_text(
        "sms:\n  aliyun:\n    access_key_id: from-conf-local\n", encoding="utf-8"
    )
    (app / "config.local.yaml").write_text(
        "sms:\n  aliyun:\n    access_key_id: \"\"\nrunAllStartEnabled: true\n",
        encoding="utf-8",
    )
    import conf_loader as cl

    monkeypatch.setattr(cl, "monorepo_root", lambda: root)
    monkeypatch.setattr(cl, "_load_addressing_scheme", lambda: object())
    monkeypatch.setattr(cl.TemplateResolver, "resolve", staticmethod(lambda cfg, scheme: cfg))
    got = cl.load_app_config("core/sms")
    assert got["sms"]["aliyun"]["access_key_id"] == "from-conf-local"
    assert "runAllStartEnabled" not in got


def test_deep_merge_list_of_maps_by_index():
    out = deep_merge(
        {"oidc": {"bootstrapClients": [{"clientId": "a", "clientSecret": ""}, {"clientId": "b", "clientSecret": ""}]}},
        {"oidc": {"bootstrapClients": [{"clientSecret": "sa"}, {"clientSecret": "sb"}]}},
    )
    assert out["oidc"]["bootstrapClients"][0] == {"clientId": "a", "clientSecret": "sa"}
    assert out["oidc"]["bootstrapClients"][1] == {"clientId": "b", "clientSecret": "sb"}
