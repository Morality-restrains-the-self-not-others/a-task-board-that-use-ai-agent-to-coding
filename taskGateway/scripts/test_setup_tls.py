#!/usr/bin/env python3
"""setup_tls stages gateway PEMs from conf-local; DEPLOY_MODE fail-closed."""

from __future__ import annotations

import os
import subprocess
from pathlib import Path

ROOT = Path(__file__).resolve().parents[1]
RUN_SH = ROOT / "run.sh"


def _setup_tls_body(text: str) -> str:
    start = text.find("setup_tls()")
    assert start >= 0
    end = text.find("\nroutes_apply()", start)
    assert end > start
    return text[start:end]


def test_setup_tls_prefers_conf_local_and_fail_closed():
    body = _setup_tls_body(RUN_SH.read_text(encoding="utf-8"))
    assert "conf-local/gateway/task-gateway" in body
    assert "dev-gateway.pem" in body
    assert "dev-gateway-key.pem" in body
    assert "DEPLOY_MODE" in body
    assert "INFRA_HOST" in body
    assert "10.2.150.89" not in body


def test_setup_tls_stages_from_conf_local(tmp_path):
    dest = tmp_path / "gw"
    dest.mkdir()
    (dest / "run.sh").write_bytes(RUN_SH.read_bytes())
    (dest / "run.sh").chmod(0o755)
    deploy = tmp_path / "deploy"
    pem_dir = deploy / "conf-local" / "gateway" / "task-gateway"
    pem_dir.mkdir(parents=True)
    (pem_dir / "dev-gateway.pem").write_text("CERT\n", encoding="utf-8")
    (pem_dir / "dev-gateway-key.pem").write_text("KEY\n", encoding="utf-8")
    env = {k: v for k, v in os.environ.items() if k not in {"DEPLOY_ROOT", "CONF_ROOT"}}
    env["DEPLOY_ROOT"] = str(deploy)
    env["DEPLOY_MODE"] = "1"
    r = subprocess.run(
        ["bash", str(dest / "run.sh"), "setup-tls"],
        env=env,
        capture_output=True,
        text=True,
        check=False,
    )
    assert r.returncode == 0, r.stderr + r.stdout
    assert (dest / "certs" / "dev-gateway.pem").read_text(encoding="utf-8") == "CERT\n"
    assert (dest / "certs" / "dev-gateway-key.pem").read_text(encoding="utf-8") == "KEY\n"


def test_setup_tls_deploy_mode_fails_without_pem(tmp_path):
    dest = tmp_path / "gw"
    dest.mkdir()
    (dest / "run.sh").write_bytes(RUN_SH.read_bytes())
    (dest / "run.sh").chmod(0o755)
    env = {k: v for k, v in os.environ.items() if k not in {"DEPLOY_ROOT", "CONF_ROOT"}}
    env["DEPLOY_ROOT"] = str(tmp_path / "empty-deploy")
    env["DEPLOY_MODE"] = "1"
    (tmp_path / "empty-deploy").mkdir()
    r = subprocess.run(
        ["bash", str(dest / "run.sh"), "setup-tls"],
        env=env,
        capture_output=True,
        text=True,
        check=False,
    )
    assert r.returncode != 0
    assert "conf-local" in (r.stderr + r.stdout)
