#!/usr/bin/env python3
"""回归单测：upstream healthCheck → APISIX active checks 发射（OPT-20260811-008）。

- healthCheck 键存在时生成 checks.active（type/http_path/超时/健康/不健康阈值）；
- 未配置 healthCheck 的上游不生成 checks（保持既有 upstream 形状，防止全站 502）；
- http_path 非 / 开头时硬失败（防生成非法 APISIX 配置）；
- host 为空时不发射 host 字段（APISIX schema 拒绝空串，见 reload 验证）。
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


def test_health_check_emitted_for_configured_upstream():
    up = _upstream(healthCheck={"http_path": "/api/health/"})
    out = m._upstream_health_checks(up)
    active = out["checks"]["active"]
    assert active["type"] == "http"
    assert active["http_path"] == "/api/health/"
    assert active["timeout"] == 3
    assert active["healthy"] == {"interval": 5, "successes": 2}
    assert active["unhealthy"] == {"interval": 5, "http_failures": 3}
    assert "host" not in active  # 未配置探针 Host 时不发射空 host


def test_health_check_custom_values_preserved():
    up = _upstream(
        healthCheck={
            "http_path": "/healthz",
            "timeout": 2,
            "healthy_interval": 10,
            "successes": 3,
            "unhealthy_interval": 8,
            "http_failures": 5,
            "host": "taskauth.internal",
        }
    )
    active = m._upstream_health_checks(up)["checks"]["active"]
    assert active["http_path"] == "/healthz"
    assert active["timeout"] == 2
    assert active["healthy"] == {"interval": 10, "successes": 3}
    assert active["unhealthy"] == {"interval": 8, "http_failures": 5}
    assert active["host"] == "taskauth.internal"


def test_no_checks_without_health_check_key():
    up = _upstream()
    assert m._upstream_health_checks(up) == {}
    assert m._upstream_health_checks(None) == {}
    assert m._upstream_health_checks({"host": "127.0.0.1", "docs": False}) == {}


def test_http_path_must_start_with_slash():
    up = _upstream(healthCheck={"http_path": "api/health/"})
    try:
        m._upstream_health_checks(up)
    except ValueError as e:
        assert "http_path" in str(e)
        return
    raise AssertionError("http_path 非 / 开头必须被拒绝（未抛 ValueError）")


def test_generated_apisix_contains_task_auth_checks():
    """产物门禁：up-taskAuth 必须含 active checks，其余上游不受影响。"""
    text = m.generate(check_only=False, allow_write=False)
    up_task_auth = "checks:\n    active:\n      type: http\n      http_path: /api/health/"
    assert up_task_auth in text, "up-taskAuth 缺少 active health checks"


if __name__ == "__main__":
    tests = [v for k, v in sorted(globals().items()) if k.startswith("test_")]
    for t in tests:
        t()
        print(f"PASS {t.__name__}")
    print(f"OK — {len(tests)} tests passed")
