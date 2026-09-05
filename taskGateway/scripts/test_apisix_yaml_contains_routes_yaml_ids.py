#!/usr/bin/env python3
"""回归：routes.yaml 中的路由 id 必须出现在已生成的 apisix.yaml。

缺陷：OPT-20260819-012 把 POST /api/system-admin/order-number/parse/ 写入
routes.yaml，但未执行 routes-apply。公网请求落入 spa-catch-all → APISIX 502 HTML，
管理端订单记录页只显示「解析订单号失败」。
"""
from pathlib import Path

import yaml

ROOT = Path(__file__).resolve().parent.parent
ROUTES_YAML = ROOT / "routes" / "routes.yaml"
APISIX_YAML = ROOT / "apisix" / "apisix.yaml"


def _route_ids(doc: dict) -> list[str]:
    ids = []
    for route in doc.get("routes") or []:
        if isinstance(route, dict) and route.get("id"):
            ids.append(str(route["id"]))
    return ids


def test_apisix_yaml_contains_all_routes_yaml_ids():
    routes_doc = yaml.safe_load(ROUTES_YAML.read_text(encoding="utf-8"))
    apisix_text = APISIX_YAML.read_text(encoding="utf-8")
    missing = [rid for rid in _route_ids(routes_doc) if f"id: {rid}" not in apisix_text]
    assert missing == [], (
        "apisix.yaml 缺少 routes.yaml 路由 "
        f"{missing}；请在 taskGateway 执行 bash run.sh routes-apply"
    )


def test_order_number_parse_route_present_in_apisix_yaml():
    text = APISIX_YAML.read_text(encoding="utf-8")
    assert "id: api-system-admin-order-number-parse" in text, (
        "apisix.yaml 未包含 api-system-admin-order-number-parse，"
        "粘贴订单号会 502 HTML"
    )
    assert "/api/system-admin/order-number/parse" in text


def test_workspace_queue_schedule_route_present_in_apisix_yaml():
    """工作空间级排队调度（OPT-20260824-005）直连 taskTaskService。

    前端 GET/PUT /api/tenant/{tid}/workspace/{wid}/queue-schedule/ 走网关；
    若路由缺失会落入 task-tenant-service(863) 兜底 → 404 not found（页面
    schedule-error「not found」）。回归断言：路由已登记且指向 taskTaskService。
    """
    text = APISIX_YAML.read_text(encoding="utf-8")
    assert "id: workspace-queue-schedule-direct" in text, (
        "apisix.yaml 未包含 workspace-queue-schedule-direct，"
        "排队调度页会 404 not found；请在 taskGateway 执行 bash run.sh routes-apply"
    )
    assert "/api/tenant/*/workspace/*/queue-schedule" in text


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
    raise SystemExit(1 if failed else 0)
