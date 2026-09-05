"""mysqld-exporter v0.16 rejects --mysqld.password and needs a mounted my.cnf.

Without that, job mysqld-exporter stays down and 数据库分区状态监控 is empty.
"""

from __future__ import annotations

from pathlib import Path

import yaml


def test_mysqld_exporter_uses_mounted_mycnf_not_removed_flags() -> None:
    root = Path(__file__).resolve().parents[1]
    data = yaml.safe_load((root / "docker-compose.yaml").read_text(encoding="utf-8"))
    svc = data["services"]["mysqld-exporter"]
    cmd = svc.get("command") or []
    joined = " ".join(str(c) for c in cmd)
    assert "--mysqld.password" not in joined
    assert "--mysqld.username" not in joined
    assert "--config.my-cnf=/etc/mysqld-exporter/my.cnf" in joined
    volumes = svc.get("volumes") or []
    assert any("mysqld-exporter/my.cnf" in str(v) for v in volumes)
    cnf = (root / "mysqld-exporter" / "my.cnf").read_text(encoding="utf-8")
    assert "user=" in cnf
    assert "host.docker.internal" in cnf
    assert any(str(c).startswith("--collect.") for c in cmd)
