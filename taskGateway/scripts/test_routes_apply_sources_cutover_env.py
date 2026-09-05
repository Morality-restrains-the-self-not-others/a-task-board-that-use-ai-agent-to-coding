#!/usr/bin/env python3
"""OPT-20260901-006: routes_apply 在生成 apisix.yaml 前须 source 部署根 cutover.env。

clone-run 的 taskGateway 在 envs/current/，机密 conf-local 在部署根。只靠向上 walk 时，
中间目录若误放非空 conf-local 会提前停（密钥错根 → HTTP 200、密钥空、厂商 AUTH 失败）。
routes_apply 须显式 resolve_deploy_root 并 export DEPLOY_ROOT/CONF_ROOT。
"""
from __future__ import annotations

import os
import subprocess
from pathlib import Path

ROOT = Path(__file__).resolve().parents[1]
RUN_SH = ROOT / "run.sh"


def _routes_apply_body(text: str) -> str:
    start = text.find("routes_apply()")
    assert start >= 0
    end = text.find("\n}", start)
    assert end > start
    return text[start:end]


def _setup_tls_body(text: str) -> str:
    start = text.find("setup_tls()")
    assert start >= 0
    end = text.find("\nresolve_deploy_root", start)
    # setup_tls 定义在 resolve_deploy_root 之后；向后找 routes_apply 之前即可
    if end < 0:
        end = text.find("\nroutes_apply()", start)
    assert end > start
    return text[start:end]


def test_routes_apply_sources_cutover_env_and_exports_deploy_root() -> None:
    body = _routes_apply_body(RUN_SH.read_text(encoding="utf-8"))
    assert "resolve_deploy_root" in body, "routes_apply 必须先解析部署根"
    assert "cutover.env" in body, "routes_apply 必须 source 部署根 cutover.env"
    assert 'export DEPLOY_ROOT="$deploy_root"' in body, "必须显式导出 DEPLOY_ROOT"
    assert "export CONF_ROOT=" in body, "必须显式导出 CONF_ROOT"


def test_setup_tls_reuses_resolve_deploy_root() -> None:
    body = _setup_tls_body(RUN_SH.read_text(encoding="utf-8"))
    assert "resolve_deploy_root" in body, "setup_tls 应复用 resolve_deploy_root，去掉内联 walk"


def _resolve_deploy_root(env: dict[str, str], dest: Path) -> str:
    (dest / "run.sh").write_bytes(RUN_SH.read_bytes())
    (dest / "run.sh").chmod(0o755)
    e = {k: v for k, v in os.environ.items() if k not in {"DEPLOY_ROOT", "CONF_ROOT"}}
    e.update(env)
    r = subprocess.run(
        ["bash", str(dest / "run.sh"), "resolve-deploy-root"],
        env=e,
        capture_output=True,
        text=True,
        check=False,
    )
    assert r.returncode == 0, r.stderr + r.stdout
    return r.stdout.strip()


def test_resolve_deploy_root_walks_to_conf_base_yaml(tmp_path: Path) -> None:
    dest = tmp_path / "gw"
    dest.mkdir()
    # 源码仓场景：run.sh 在 <root>/taskGateway，conf/base.yaml 在 <root>/conf
    (tmp_path / "conf" / "base.yaml").parent.mkdir(parents=True)
    (tmp_path / "conf" / "base.yaml").write_text("scheme: https\n", encoding="utf-8")
    got = _resolve_deploy_root({}, dest)
    assert got == str(tmp_path), f"walk 应上溯到含 conf/base.yaml 的根，实际 {got!r}"


def test_resolve_deploy_root_prefers_dep_root_env(tmp_path: Path) -> None:
    dest = tmp_path / "gw"
    dest.mkdir()
    got = _resolve_deploy_root({"DEPLOY_ROOT": "/home/ljy/bin/daydaymoney-deploy"}, dest)
    assert got == "/home/ljy/bin/daydaymoney-deploy"


def test_resolve_deploy_root_derives_from_conf_root(tmp_path: Path) -> None:
    dest = tmp_path / "gw"
    dest.mkdir()
    root = tmp_path / "deploy"
    (root / "conf").mkdir(parents=True)
    (root / "conf" / "base.yaml").write_text("scheme: https\n", encoding="utf-8")
    got = _resolve_deploy_root({"CONF_ROOT": str(root / "conf")}, dest)
    assert got == str(root), f"CONF_ROOT=<root>/conf 时应取 <root>，实际 {got!r}"


if __name__ == "__main__":
    import tempfile

    failed = 0
    for name, fn in sorted(globals().items()):
        if not name.startswith("test_") or not callable(fn):
            continue
        with tempfile.TemporaryDirectory() as td:
            try:
                fn(Path(td))
                print(f"PASS {name}")
            except Exception as e:  # noqa: BLE001
                failed += 1
                print(f"FAIL {name}: {e}")
    raise SystemExit(1 if failed else 0)
