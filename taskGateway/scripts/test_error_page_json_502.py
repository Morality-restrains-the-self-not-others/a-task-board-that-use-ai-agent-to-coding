#!/usr/bin/env python3
"""回归：APISIX 上游 502/503/504 必须返回 JSON 错误体而非 HTML。

缺陷：OPT-20260825-018 前，taskBill 重启窗口 APISIX `connect() failed (111)`
返回 `Content-Type: text/html` 的 502，前端 `response.json()` 解析失败，
只能落到无信息兜底文案。修复在 `apisix/config.yaml` 的
`nginx_config.http_server_configuration_snippet`：error_page 502/503/504
→ 命名 location 输出 JSON（`error` + `trace_id`）并保留 `X-Trace-Id` 响应头。
"""
from pathlib import Path

import yaml

ROOT = Path(__file__).resolve().parent.parent
CONFIG_YAML = ROOT / "apisix" / "config.yaml"


def _nginx_snippet() -> str:
    doc = yaml.safe_load(CONFIG_YAML.read_text(encoding="utf-8"))
    snippet = (doc.get("nginx_config") or {}).get("http_server_configuration_snippet")
    assert isinstance(snippet, str) and snippet.strip(), (
        "apisix/config.yaml 缺少 nginx_config.http_server_configuration_snippet；"
        "上游 502 会退回 HTML 错误页（见 OPT-20260825-018）"
    )
    return snippet


def test_error_page_covers_502_503_504():
    snippet = _nginx_snippet()
    assert "error_page 502 503 504 = @gateway_error;" in snippet, (
        "error_page 需覆盖 502/503/504 并重定向到 @gateway_error；"
        f"实际 snippet:\n{snippet}"
    )


def test_error_page_returns_json_body_with_trace():
    snippet = _nginx_snippet()
    assert "default_type application/json;" in snippet, (
        "命名 location 须声明 application/json，否则 return 字符串按 text/plain 发送"
    )
    assert "服务暂时不可用，请稍后重试" in snippet, "JSON body 应含用户可读 error 文案"
    assert '"trace_id":"$http_x_trace_id"' in snippet, (
        "JSON body 应带 trace_id（代理层错误唯一来源是请求 X-Trace-Id）"
    )


def test_error_page_preserves_x_trace_id_response_header():
    snippet = _nginx_snippet()
    assert "add_header X-Trace-Id $http_x_trace_id;" in snippet, (
        "响应应保留 X-Trace-Id 头供前端 data-traceId 关联"
    )


def test_error_page_named_location_defined():
    snippet = _nginx_snippet()
    assert "location @gateway_error {" in snippet, (
        "error_page 引用的命名 location @gateway_error 必须在本 snippet 内定义"
    )


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
