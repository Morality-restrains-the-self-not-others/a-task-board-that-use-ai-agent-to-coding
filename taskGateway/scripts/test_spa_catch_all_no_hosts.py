#!/usr/bin/env python3
"""回归单测：spa-catch-all 禁止绑定 hosts（OPT-20260809-006）。

缺陷：回归点 8f6194c（08-03）给 spa-catch-all（priority 10, uri /*, GET/HEAD）
绑定 hosts [daydaymoney.com, www.daydaymoney.com] 后，该域名下全部 GET/HEAD——
包括 /api/auth/* ——被 host 匹配分支优先命中并转发到 taskFE(4000) → 500/502，
微信登录静默挂 6 天（08-03→08-09）。修复（8734eb2）已移除 hosts，但仅靠注释
防护不可靠。本测试守护生成器静态门禁：spa-catch-all 只要声明 hosts 字段即
硬失败（--check/--lint/写出路径均触发），防止 hosts 回归重新引入。
"""
import importlib.util
import sys
from pathlib import Path

import yaml

SCRIPT_DIR = Path(__file__).resolve().parent  # taskGateway/scripts/
GENERATOR = SCRIPT_DIR / "routes-to-apisix.py"
ROOT = SCRIPT_DIR.parent  # taskGateway/
APISIX_YAML = ROOT / "apisix" / "apisix.yaml"
ROUTES_YAML = ROOT / "routes" / "routes.yaml"

# routes-to-apisix.py 依赖同目录 sibling 模块（cors_allow_headers.py）
sys.path.insert(0, str(SCRIPT_DIR))

_spec = importlib.util.spec_from_file_location("routes_to_apisix", GENERATOR)
m = importlib.util.module_from_spec(_spec)
_spec.loader.exec_module(m)  # type: ignore[union-attr]


def _spa_route(**overrides) -> dict:
    rule = {
        "id": "spa-catch-all",
        "priority": 10,
        "uri": "/*",
        "methods": ["GET", "HEAD"],
        "upstream": "taskFE",
        "auth_mode": "public",
    }
    rule.update(overrides)
    return rule


def test_spa_catch_all_with_hosts_raises():
    """spa-catch-all 声明 hosts 字段 → 静态门禁必须 ValueError 硬失败。"""
    routes_doc = {"routes": [_spa_route(hosts=["${subdomains.base}"])]}
    try:
        m.validate_spa_catch_all_no_hosts(routes_doc)
    except ValueError as e:
        assert "spa-catch-all" in str(e) and "OPT-20260809-006" in str(e)
        return
    raise AssertionError("spa-catch-all 带 hosts 必须被拒绝（未抛 ValueError）")


def test_spa_catch_all_empty_hosts_list_raises():
    """hosts 空列表也算声明了该字段（APISIX 仍会走 host 匹配分支）→ 同样拒绝。"""
    routes_doc = {"routes": [_spa_route(hosts=[])]}
    try:
        m.validate_spa_catch_all_no_hosts(routes_doc)
    except ValueError:
        return
    raise AssertionError("spa-catch-all 带空 hosts 列表也必须被拒绝")


def test_spa_catch_all_without_hosts_passes():
    """spa-catch-all 不声明 hosts → 放行（当前正确形态，不许误伤）。"""
    routes_doc = {"routes": [_spa_route()]}
    m.validate_spa_catch_all_no_hosts(routes_doc)  # 不抛即为通过


def test_other_route_hosts_allowed():
    """非 spa-catch-all 路由绑定 hosts 不受此门禁影响（门禁只守 catch-all）。"""
    routes_doc = {
        "routes": [
            {"id": "billing-tenant-direct", "uri": "/api/billing/*", "hosts": ["${subdomains.base}"]},
        ]
    }
    m.validate_spa_catch_all_no_hosts(routes_doc)


def _min_routes_doc(**route_overrides) -> dict:
    """构造能通过 generate() 前置校验（installed-images 等）的最小 routes_doc。"""
    spa = _spa_route(**route_overrides)
    return {
        "routes": [
            # 前置校验 validate_installed_images_route_to_task_cloud 要求 task-cloud-service
            {"id": "task-cloud-service", "priority": 850, "uris": ["/api/cloud/*"], "upstream": "taskCloudService"},
            spa,
        ],
        "upstreams": {},
    }


def test_generate_check_rejects_hosts_regression():
    """端到端：generate()（--check 路径）在 routes.yaml 注入 hosts 回归时必须抛
    ValueError（模拟 8f6194c 回归点，验证门禁挂在生成链最前、不可绕过）。"""
    original_load = m._load
    bad_doc = _min_routes_doc(hosts=["${subdomains.base}", "${subdomains.www}"])
    m._load = lambda: (bad_doc, {})
    try:
        try:
            m.generate(check_only=True, lint_strict=False)
        except ValueError as e:
            assert "spa-catch-all" in str(e) and "OPT-20260809-006" in str(e), f"非预期错误: {e}"
            return
        raise AssertionError("generate() 未拦截 spa-catch-all hosts 回归（门禁缺失/被绕过）")
    finally:
        m._load = original_load


def test_generated_apisix_spa_catch_all_no_hosts():
    """端到端守护产物：当前 apisix.yaml 的 spa-catch-all 路由不得含 hosts 字段。"""
    doc = yaml.safe_load(APISIX_YAML.read_text(encoding="utf-8"))
    target = None
    for route in doc.get("routes", []):
        if route.get("id") == "spa-catch-all":
            target = route
            break
    assert target is not None, "apisix.yaml 未找到 spa-catch-all 路由（测试前提失效）"
    assert "hosts" not in target, (
        f"spa-catch-all 在 apisix.yaml 中仍含 hosts={target.get('hosts')} —— "
        "回归 8f6194c 复现，生成门禁未生效"
    )


def test_real_routes_file_passes_gate():
    """当前 routes.yaml 真源必须通过门禁（当前正确形态，防误伤）。"""
    routes_doc = yaml.safe_load(ROUTES_YAML.read_text(encoding="utf-8"))
    m.validate_spa_catch_all_no_hosts(routes_doc)


def test_lint_mode_does_not_write_outputs():
    """--lint 路径（allow_write=False）不得写出 apisix.yaml / gatewaycors 产物。

    OPT-20260809-006 连带修复：此前 --lint 实际会写文件（注释称「不写文件」但代码
    未兑现），在无 docker 环境写出 loopback 上游节点 → bind-mount 热载 → 全站 502
    （与 OPT-20260806-026 同类事故）。本测试用临时 OUT 验证 lint 只校验不落盘。
    """
    import tempfile
    from pathlib import Path

    original_out, original_cors = m.OUT, m.GATEWAYCORS_OUT
    tmpdir = Path(tempfile.mkdtemp(prefix="opt-gate-lint-"))
    tmp_out, tmp_cors = tmpdir / "apisix.yaml", tmpdir / "allow_headers_gen.go"
    m.OUT, m.GATEWAYCORS_OUT = tmp_out, tmp_cors
    try:
        routes_doc = _min_routes_doc()  # spa-catch-all 无 hosts（合法形态）
        original_load = m._load
        m._load = lambda: (routes_doc, {})
        try:
            text = m.generate(check_only=False, lint_strict=False, allow_write=False)
        finally:
            m._load = original_load
        assert text and "#END" in text, "generate 应返回生成文本"
        assert not tmp_out.exists(), f"--lint 不应写出 {tmp_out}"
        assert not tmp_cors.exists(), f"--lint 不应写出 {tmp_cors}"
    finally:
        m.OUT, m.GATEWAYCORS_OUT = original_out, original_cors
        import shutil

        shutil.rmtree(tmpdir, ignore_errors=True)


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
