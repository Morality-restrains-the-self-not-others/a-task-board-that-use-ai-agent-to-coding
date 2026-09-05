#!/usr/bin/env python3
"""回归单测：forward-auth 上游头白名单必须透传 X-User-Email（OPT-20260807-010）。

缺陷：taskAuth 的 writeForwardAuthHeaders 已注入 X-User-Email（主邮箱，供下游
厂商门户资格判断等邮箱匹配），但 APISIX forward-auth 插件 upstream_headers
白名单缺少该头 → 网关静默丢弃 → taskAiProvider 永远看到空邮箱：
  - vendor-status has_email=!isSyntheticEmail("")=true（表单能打开）
  - vendor-application email=="" → 400「厂商门户需先绑定邮箱账号」。
用户已绑定邮箱仍报此错，根因在网关层丢头，本测试守护生成器与产物双重一致性。
"""
import importlib.util
import sys
from pathlib import Path

import yaml

SCRIPT_DIR = Path(__file__).resolve().parent  # taskGateway/scripts/
GENERATOR = SCRIPT_DIR / "routes-to-apisix.py"
ROOT = SCRIPT_DIR.parent.parent
APISIX_YAML = ROOT / "taskGateway" / "apisix" / "apisix.yaml"

# routes-to-apisix.py 依赖同目录 sibling 模块（cors_allow_headers.py）
sys.path.insert(0, str(SCRIPT_DIR))

_spec = importlib.util.spec_from_file_location("routes_to_apisix", GENERATOR)
m = importlib.util.module_from_spec(_spec)
_spec.loader.exec_module(m)  # type: ignore[union-attr]


def _min_conf() -> dict:
    return {
        "auth": {"taskauthInternalSecret": "test-secret"},
        "gatewayInternalSecret": "test-secret",
        # _upstream_host 在 TASK_GATEWAY_APISIX_IN_DOCKER=1 下要求 docker.upstreamHost
        # （CI 与本地环境均可能设置该变量），显式提供使测试与环境无关。
        "docker": {"upstreamHost": "host.docker.internal"},
    }


def _min_upstreams() -> dict:
    return {"taskAuth": {"host": "127.0.0.1", "port": 8003}}


def test_forward_auth_plugin_forwards_email_header():
    """生成器产出的 forward-auth 插件必须把 X-User-Email 列入 upstream_headers。"""
    plugin = m._forward_auth_plugin(_min_conf(), _min_upstreams())
    upstream_headers = plugin["forward-auth"]["upstream_headers"]
    assert "X-User-Email" in upstream_headers, (
        "upstream_headers 缺少 X-User-Email：网关将丢弃 taskAuth 注入的主邮箱头，"
        "下游邮箱匹配（厂商门户 vendor-status / vendor-application）永远失败"
    )


def test_generated_apisix_all_forward_auth_blocks_forward_email():
    """生成的 apisix.yaml 中每个 forward-auth 块的 upstream_headers 都必须含 X-User-Email
    （端到端守护：任何路由注册遗漏该头都会重新踩中 OPT-20260807-010 缺陷）。"""
    doc = yaml.safe_load(APISIX_YAML.read_text(encoding="utf-8"))
    blocks = 0
    for route in doc.get("routes", []):
        fa = (route.get("plugins") or {}).get("forward-auth")
        if fa is None:
            continue
        blocks += 1
        assert "X-User-Email" in fa.get("upstream_headers", []), (
            f"route {route.get('id')} 的 forward-auth upstream_headers 缺少 X-User-Email"
        )
    assert blocks > 0, "apisix.yaml 中未发现任何 forward-auth 块（测试前提失效）"


if __name__ == "__main__":
    import traceback

    failed = 0
    for name, fn in sorted(globals().items()):
        if name.startswith("test_") and callable(fn):
            try:
                fn()
                print(f"PASS {name}")
            except AssertionError as e:
                failed += 1
                print(f"FAIL {name}: {e}")
                traceback.print_exc()
    sys.exit(1 if failed else 0)
