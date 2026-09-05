#!/usr/bin/env python3
"""OPT-20260901-005: clone-run(DEPLOY_MODE/DEPLOY_ROOT) 与源码仓隔离 compose 项目名。

同机双根时 docker compose 默认项目名都是 taskgateway，`compose down` 会拆掉另一棵树的
容器。DEPLOY_MODE=1 或 DEPLOY_ROOT 存在时须用 taskgateway-deploy，APISIX 容器名跟随；
显式 COMPOSE_PROJECT_NAME 始终优先。
"""
from __future__ import annotations

import os
import subprocess
from pathlib import Path

ROOT = Path(__file__).resolve().parents[1]
RUN_SH = ROOT / "run.sh"

_CLEAR = {"DEPLOY_ROOT", "CONF_ROOT", "DEPLOY_MODE", "COMPOSE_PROJECT_NAME"}


def _identities(env: dict[str, str], dest: Path) -> str:
    (dest / "run.sh").write_bytes(RUN_SH.read_bytes())
    (dest / "run.sh").chmod(0o755)
    e = {k: v for k, v in os.environ.items() if k not in _CLEAR}
    e.update(env)
    r = subprocess.run(
        ["bash", str(dest / "run.sh"), "runtime-identities"],
        env=e,
        capture_output=True,
        text=True,
        check=False,
    )
    assert r.returncode == 0, r.stderr + r.stdout
    return r.stdout


def _project(text: str) -> str:
    return next(
        l.split("=", 1)[1].strip()
        for l in text.splitlines()
        if l.startswith("COMPOSE_PROJECT_NAME=")
    )


def _container(text: str) -> str:
    return next(
        l.split("=", 1)[1].strip()
        for l in text.splitlines()
        if l.startswith("TASKGATEWAY_APISIX_CONTAINER=")
    )


def test_source_repo_defaults_to_taskgateway(tmp_path: Path) -> None:
    text = _identities({}, tmp_path)
    assert _project(text) == "taskgateway"
    assert _container(text) == "taskgateway-apisix-1"


def test_deploy_mode_uses_isolated_project_name(tmp_path: Path) -> None:
    text = _identities({"DEPLOY_MODE": "1"}, tmp_path)
    assert _project(text) == "taskgateway-deploy"
    assert _container(text) == "taskgateway-deploy-apisix-1"


def test_deploy_root_uses_isolated_project_name(tmp_path: Path) -> None:
    text = _identities({"DEPLOY_ROOT": "/home/ljy/bin/daydaymoney-deploy"}, tmp_path)
    assert _project(text) == "taskgateway-deploy"


def test_seed_and_source_project_names_differ(tmp_path: Path) -> None:
    src = _identities({}, tmp_path)
    seed = _identities({"DEPLOY_MODE": "1"}, tmp_path)
    assert _project(src) != _project(seed), (
        "源码仓与 clone-run 项目名必须不同，否则 compose down 会误拆另一棵树容器"
    )


def test_explicit_compose_project_name_wins(tmp_path: Path) -> None:
    text = _identities({"DEPLOY_MODE": "1", "COMPOSE_PROJECT_NAME": "mygw"}, tmp_path)
    assert _project(text) == "mygw"
    assert _container(text) == "mygw-apisix-1"


def test_no_stale_hardcoded_container_name_in_run_sh(tmp_path: Path | None = None) -> None:
    text = RUN_SH.read_text(encoding="utf-8")
    assert "docker exec taskgateway-apisix-1" not in text, (
        "APISIX 容器名必须由 COMPOSE_PROJECT_NAME 派生，禁止硬编码 taskgateway-apisix-1"
    )
    assert "grep -qx 'taskgateway-apisix-1'" not in text


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
