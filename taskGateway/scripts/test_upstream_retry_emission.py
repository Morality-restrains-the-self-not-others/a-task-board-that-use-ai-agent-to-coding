#!/usr/bin/env python3
"""回归单测：upstream retry → APISIX retries/retry_timeout 发射（OPT-20260827-039）。

- `retry` 键存在时生成 `retries`（整数）与 `retry_timeout`（秒，APISIX balancer 与
  ngx.now() 相加作重试截止，见 balancer.lua `ctx.proxy_retry_deadline`）；
- 未配置 `retry` 的上游不生成 retry 字段（保持既有 upstream 形状，防止全站行为变化）；
- 生成产物：up-taskCloudService / up-taskSse / up-taskAgentSupport 必须含 retries，
  其余上游不受影响。
"""
import importlib.util
import sys
from pathlib import Path

SCRIPT_DIR = Path(__file__).resolve().parent
GENERATOR = SCRIPT_DIR / "routes-to-apisix.py"

# routes-to-apisix.py 依赖同目录 sibling 模块（cors_allow_headers.py）
sys.path.insert(0, str(SCRIPT_DIR))

_spec = importlib.util.spec_from_file_location("routes_to_apisix", GENERATOR)
m = importlib.util.module_from_spec(_spec)
_spec.loader.exec_module(m)  # type: ignore[union-attr]


def _upstream(**overrides) -> dict:
    up = {"host": "127.0.0.1", "port": 8003, "docs": False}
    up.update(overrides)
    return up


def test_retry_emitted_for_configured_upstream():
    up = _upstream(retry={"retries": 2, "retry_timeout": 3})
    out = m._upstream_retries(up)
    assert out == {"retries": 2, "retry_timeout": 3}


def test_retry_partial_keys_preserved():
    up = _upstream(retry={"retries": 1})
    out = m._upstream_retries(up)
    assert out == {"retries": 1}
    assert "retry_timeout" not in out


def test_no_retry_without_key():
    assert m._upstream_retries(_upstream()) == {}
    assert m._upstream_retries(None) == {}
    assert m._upstream_retries({"host": "127.0.0.1", "docs": False}) == {}


def test_generated_apisix_contains_compute_retries():
    """产物门禁：计算族上游含 retries，其余上游不受影响。"""
    generated = m.generate(allow_write=False)
    text = generated

    assert 'id: up-taskCloudService' in text
    cloud_block = text.split("id: up-taskCloudService", 1)[1].split("- id:", 1)[0]
    assert "retries: 2" in cloud_block
    assert "retry_timeout: 3" in cloud_block

    sse_block = text.split("id: up-taskSse", 1)[1].split("- id:", 1)[0]
    assert "retries: 2" in sse_block

    support_block = text.split("id: up-taskAgentSupport", 1)[1].split("- id:", 1)[0]
    assert "retries: 2" in support_block

    # 未配置 retry 的上游（taskAuth）不应含 retries
    auth_block = text.split("id: up-taskAuth", 1)[1].split("- id:", 1)[0]
    assert "retries:" not in auth_block
