#!/usr/bin/env python3
"""routes-to-apisix 必须合并 conf-local 的 gatewayInternalSecret。

缺陷：生成器只读已跟踪 conf/gateway/task-gateway/config.yaml（空骨架）。
clone-run 在 conf-local 写入真实密钥后，routes-apply 把
X-TaskGateway-Internal-Secret 写成 ''。APISIX 注入空头，task-project-service
ApplyGatewayUser 校验失败 → GET /api/projects/tenant_id/{tid} 401「请先登录」，
页面「加载项目失败」（traceId 7af49e7f-8e9e-4686-9470-dd7620ef4e6d）。
"""
from __future__ import annotations

import importlib.util
import os
import sys
from pathlib import Path

SCRIPT_DIR = Path(__file__).resolve().parent
GENERATOR = SCRIPT_DIR / "routes-to-apisix.py"
sys.path.insert(0, str(SCRIPT_DIR))

_spec = importlib.util.spec_from_file_location("routes_to_apisix", GENERATOR)
m = importlib.util.module_from_spec(_spec)
_spec.loader.exec_module(m)  # type: ignore[union-attr]


def _write_gateway_conf(root: Path, *, tracked: str, local: str | None) -> None:
    tracked_path = root / "conf" / "gateway" / "task-gateway" / "config.yaml"
    tracked_path.parent.mkdir(parents=True, exist_ok=True)
    tracked_path.write_text(
        f'gatewayInternalSecret: "{tracked}"\n',
        encoding="utf-8",
    )
    if local is None:
        return
    local_path = root / "conf-local" / "gateway" / "task-gateway" / "config.yaml"
    local_path.parent.mkdir(parents=True, exist_ok=True)
    local_path.write_text(
        f'gatewayInternalSecret: "{local}"\n',
        encoding="utf-8",
    )


def test_load_gateway_conf_merges_conf_local(tmp_path: Path) -> None:
    _write_gateway_conf(tmp_path, tracked="", local="from-conf-local")
    conf = m.load_gateway_conf(tmp_path)
    assert conf.get("gatewayInternalSecret") == "from-conf-local", (
        "load_gateway_conf 未合并 conf-local：APISIX 会注入空密钥，"
        "项目列表 401 请先登录"
    )


def test_load_gateway_conf_without_local_keeps_tracked_empty(tmp_path: Path) -> None:
    _write_gateway_conf(tmp_path, tracked="", local=None)
    conf = m.load_gateway_conf(tmp_path)
    assert str(conf.get("gatewayInternalSecret") or "").strip() == "", (
        "无 conf-local 时不得发明密钥"
    )


def test_token_route_uses_merged_conf_local_secret(tmp_path: Path) -> None:
    _write_gateway_conf(tmp_path, tracked="", local="from-conf-local")
    conf = m.load_gateway_conf(tmp_path)
    plugin = m._token_route_transform(conf)
    got = plugin["proxy-rewrite"]["headers"]["set"]["X-TaskGateway-Internal-Secret"]
    assert got == "from-conf-local", (
        f"token 路由密钥应为 conf-local 值，实际 {got!r}"
    )


def test_load_gateway_conf_walks_to_deploy_root_conf_local(tmp_path: Path) -> None:
    """clone-run: REPO=envs/current, conf-local 在部署根（再上两级）。"""
    env_cur = tmp_path / "envs" / "current"
    _write_gateway_conf(env_cur, tracked="", local=None)
    local_dir = tmp_path / "conf-local" / "gateway" / "task-gateway"
    local_dir.mkdir(parents=True)
    (local_dir / "config.yaml").write_text(
        'gatewayInternalSecret: "from-deploy-root"\n',
        encoding="utf-8",
    )
    conf = m.load_gateway_conf(env_cur)
    assert conf.get("gatewayInternalSecret") == "from-deploy-root", (
        "envs/current 无 conf-local 时必须向上找到部署根 overlay"
    )


def test_load_gateway_conf_prefers_explicit_deploy_root(tmp_path: Path) -> None:
    """OPT-20260901-006: DEPLOY_ROOT 显式指向部署根时，中间目录的错误 conf-local 不得截断。"""
    env_cur = tmp_path / "envs" / "current"
    # 中间目录放了一份非空但错误的 conf-local —— 只靠 walk 会提前停在这里
    _write_gateway_conf(env_cur, tracked="", local="wrong-whoami")
    deploy = tmp_path / "deploy"
    _write_gateway_conf(deploy, tracked="", local="from-deploy-root")
    old_env = os.environ.pop("DEPLOY_ROOT", None)
    os.environ["DEPLOY_ROOT"] = str(deploy)
    try:
        conf = m.load_gateway_conf(env_cur)
        assert conf.get("gatewayInternalSecret") == "from-deploy-root", (
            "DEPLOY_ROOT 显式设置时必须以部署根 conf-local 为准，"
            "不能被 envs/current 的错误 overlay 截断"
        )
    finally:
        if old_env is not None:
            os.environ["DEPLOY_ROOT"] = old_env
        else:
            os.environ.pop("DEPLOY_ROOT", None)


def test_load_gateway_conf_prefers_explicit_conf_root(tmp_path: Path) -> None:
    """OPT-20260901-006: CONF_ROOT 指向 <root>/conf 时 overlay 取 <root>/conf-local。"""
    root = tmp_path / "root"
    _write_gateway_conf(root, tracked="", local="from-conf-root")
    old_env = os.environ.pop("CONF_ROOT", None)
    os.environ["CONF_ROOT"] = str(root / "conf")
    try:
        conf = m.load_gateway_conf(root)
        assert conf.get("gatewayInternalSecret") == "from-conf-root"
    finally:
        if old_env is not None:
            os.environ["CONF_ROOT"] = old_env
        else:
            os.environ.pop("CONF_ROOT", None)


def test_resolve_infra_host_overlays_conf_local(tmp_path: Path) -> None:
    (tmp_path / "conf").mkdir(parents=True)
    (tmp_path / "conf" / "base.yaml").write_text(
        "scheme: https\nbaseDomain: x.test\n", encoding="utf-8"
    )
    di = tmp_path / "conf" / "infra" / "docker-infra"
    di.mkdir(parents=True)
    (di / "config.yaml").write_text("infraHost: tracked.example\n", encoding="utf-8")
    loc = tmp_path / "conf-local" / "infra" / "docker-infra"
    loc.mkdir(parents=True)
    (loc / "config.yaml").write_text("infraHost: from-conf-local\n", encoding="utf-8")
    old_conf, old_cache = m.DOCKER_INFRA_CONF, m._cached_infra_host
    old_env = os.environ.pop("INFRA_HOST", None)
    m.DOCKER_INFRA_CONF = di / "config.yaml"
    m._cached_infra_host = None
    try:
        assert m._resolve_infra_host() == "from-conf-local"
    finally:
        m.DOCKER_INFRA_CONF = old_conf
        m._cached_infra_host = old_cache
        if old_env is not None:
            os.environ["INFRA_HOST"] = old_env


if __name__ == "__main__":
    import tempfile
    import traceback

    failed = 0
    with tempfile.TemporaryDirectory() as td:
        root = Path(td)
        for name, fn in sorted(globals().items()):
            if name.startswith("test_") and callable(fn):
                try:
                    fn(root / name)
                    print(f"PASS {name}")
                except AssertionError as e:
                    failed += 1
                    print(f"FAIL {name}: {e}")
                    traceback.print_exc()
                except Exception as e:
                    failed += 1
                    print(f"ERROR {name}: {e}")
                    traceback.print_exc()
    raise SystemExit(1 if failed else 0)
