#!/usr/bin/env python3
"""forward-auth 必须把 X-Impersonator-Id 列入 upstream_headers（ADR-0037）。"""
import importlib.util
import sys
from pathlib import Path

import yaml

SCRIPT_DIR = Path(__file__).resolve().parent
GENERATOR = SCRIPT_DIR / "routes-to-apisix.py"
ROOT = SCRIPT_DIR.parent.parent
APISIX_YAML = ROOT / "taskGateway" / "apisix" / "apisix.yaml"

sys.path.insert(0, str(SCRIPT_DIR))

_spec = importlib.util.spec_from_file_location("routes_to_apisix", GENERATOR)
m = importlib.util.module_from_spec(_spec)
_spec.loader.exec_module(m)  # type: ignore[union-attr]


def _min_conf() -> dict:
    return {
        "auth": {"taskauthInternalSecret": "test-secret"},
        "gatewayInternalSecret": "test-secret",
        "docker": {"upstreamHost": "host.docker.internal"},
    }


def _min_upstreams() -> dict:
    return {"taskAuth": {"host": "127.0.0.1", "port": 8003}}


def test_forward_auth_plugin_forwards_impersonator_header():
    plugin = m._forward_auth_plugin(_min_conf(), _min_upstreams())
    upstream_headers = plugin["forward-auth"]["upstream_headers"]
    assert "X-Impersonator-Id" in upstream_headers, (
        "upstream_headers 缺少 X-Impersonator-Id：网关会丢弃模拟会话操作者头"
    )
    assert "X-Impersonation-Session-Id" in upstream_headers, (
        "upstream_headers 缺少 X-Impersonation-Session-Id：访问日志无法关联模拟会话"
    )
    assert "X-Impersonating" in upstream_headers, (
        "upstream_headers 缺少 X-Impersonating"
    )


def test_generated_apisix_impersonation_route_exists():
    doc = yaml.safe_load(APISIX_YAML.read_text(encoding="utf-8"))
    ids = {route.get("id") for route in doc.get("routes", [])}
    assert "taskauth-impersonation" in ids, "apisix.yaml 缺少 taskauth-impersonation 路由"


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
