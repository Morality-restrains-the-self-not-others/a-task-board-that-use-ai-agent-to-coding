from __future__ import annotations

from pathlib import Path

import pytest

from infra_conf_host import infra_host_from_conf


def test_literal_host(tmp_path: Path) -> None:
    p = tmp_path / "config.yaml"
    p.write_text("host: 10.9.9.9\nport: 6379\n", encoding="utf-8")
    assert infra_host_from_conf(str(p), environ={}) == "10.9.9.9"


def test_yaml_default_when_env_unset(tmp_path: Path) -> None:
    p = tmp_path / "config.yaml"
    p.write_text("host: ${INFRA_HOST:-10.8.8.8}\n", encoding="utf-8")
    assert infra_host_from_conf(str(p), environ={}) == "10.8.8.8"


def test_env_overrides_yaml_default(tmp_path: Path) -> None:
    p = tmp_path / "config.yaml"
    p.write_text("host: ${INFRA_HOST:-10.8.8.8}\n", encoding="utf-8")
    assert infra_host_from_conf(str(p), environ={"INFRA_HOST": "10.1.1.1"}) == "10.1.1.1"


def test_overlays_conf_local_host(tmp_path: Path) -> None:
    conf = tmp_path / "conf" / "infra" / "docker-redis"
    conf.mkdir(parents=True)
    (tmp_path / "conf" / "base.yaml").write_text("scheme: https\nbaseDomain: example.test\n", encoding="utf-8")
    (conf / "config.yaml").write_text("host: tracked.example\nport: 6379\n", encoding="utf-8")
    local = tmp_path / "conf-local" / "infra" / "docker-redis"
    local.mkdir(parents=True)
    (local / "config.yaml").write_text("host: from-conf-local\n", encoding="utf-8")
    assert infra_host_from_conf(str(conf / "config.yaml"), environ={}) == "from-conf-local"


def test_missing_host_raises(tmp_path: Path) -> None:
    p = tmp_path / "config.yaml"
    p.write_text("port: 6379\n", encoding="utf-8")
    with pytest.raises(ValueError, match="missing host"):
        infra_host_from_conf(str(p), environ={})
