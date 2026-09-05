#!/usr/bin/env python3
"""Smoke-test AiMonitor Loki + Tempo + Grafana for local distributed tracing."""
from __future__ import annotations

import json
import sys
import time
import urllib.error
import urllib.request
from typing import Any

TEMPO_READY = "http://127.0.0.1:3200/ready"
TEMPO_SEARCH = "http://127.0.0.1:3200/api/search"
TEMPO_OTLP_HTTP = "http://127.0.0.1:4318/v1/traces"
LOKI_READY = "http://127.0.0.1:3100/ready"
GRAFANA_HEALTH = "http://127.0.0.1:3000/api/health"

# Fixed ids so the script is idempotent in assertions.
SMOKE_TRACE_HEX = "0123456789abcdef0123456789abcdef"
SMOKE_SPAN_HEX = "0123456789abcdef"


def _get(url: str, timeout: float = 5.0) -> tuple[int, str]:
    req = urllib.request.Request(url, method="GET")
    try:
        with urllib.request.urlopen(req, timeout=timeout) as resp:
            return resp.status, resp.read().decode("utf-8", errors="replace")
    except urllib.error.HTTPError as exc:
        body = exc.read().decode("utf-8", errors="replace") if exc.fp else ""
        return exc.code, body


def _post_json(url: str, payload: dict[str, Any], timeout: float = 10.0) -> int:
    data = json.dumps(payload).encode("utf-8")
    req = urllib.request.Request(
        url,
        data=data,
        method="POST",
        headers={"Content-Type": "application/json"},
    )
    with urllib.request.urlopen(req, timeout=timeout) as resp:
        return resp.status


def wait_ready(url: str, label: str, attempts: int = 12, sleep_s: float = 2.5) -> None:
    for i in range(attempts):
        code, body = _get(url)
        if code == 200 and "ready" in body.lower():
            print(f"OK  {label} ready ({url})")
            return
        if i < attempts - 1:
            time.sleep(sleep_s)
    raise SystemExit(f"FAIL {label} not ready after {attempts} attempts: {url} last={code} {body[:120]}")


def send_smoke_span() -> None:
    now_ns = time.time_ns()
    payload = {
        "resourceSpans": [
            {
                "resource": {
                    "attributes": [
                        {"key": "service.name", "value": {"stringValue": "aimonitor-smoke"}},
                    ]
                },
                "scopeSpans": [
                    {
                        "spans": [
                            {
                                "traceId": SMOKE_TRACE_HEX,
                                "spanId": SMOKE_SPAN_HEX,
                                "name": "aimonitor-smoke-test",
                                "kind": 2,
                                "startTimeUnixNano": str(now_ns - 1_000_000),
                                "endTimeUnixNano": str(now_ns),
                            }
                        ]
                    }
                ],
            }
        ]
    }
    status = _post_json(TEMPO_OTLP_HTTP, payload)
    if status not in (200, 202):
        raise SystemExit(f"FAIL OTLP ingest HTTP {status}")
    print(f"OK  OTLP smoke span ingested traceId={SMOKE_TRACE_HEX}")


def assert_trace_searchable() -> None:
    # Tempo needs a short ingest delay before search indexes the trace.
    time.sleep(3)
    url = f"{TEMPO_SEARCH}?tags=service.name%3Daimonitor-smoke&limit=20"
    code, body = _get(url)
    if code != 200:
        raise SystemExit(f"FAIL Tempo search HTTP {code}: {body[:200]}")
    data = json.loads(body)
    traces = data.get("traces") or []
    found = any(t.get("traceID") == SMOKE_TRACE_HEX for t in traces)
    if not found:
        # trace_by_id is more reliable for smoke id
        code2, body2 = _get(
            f"http://127.0.0.1:3200/api/traces/{SMOKE_TRACE_HEX}",
            timeout=10.0,
        )
        if code2 != 200:
            raise SystemExit(
                f"FAIL smoke trace not found (search={len(traces)} traces, by-id HTTP {code2})"
            )
    print(f"OK  Tempo stores and returns smoke trace {SMOKE_TRACE_HEX}")


def main() -> int:
    print("AiMonitor distributed tracing smoke test")
    print("—" * 40)
    wait_ready(TEMPO_READY, "Tempo")
    wait_ready(LOKI_READY, "Loki")
    code, _ = _get(GRAFANA_HEALTH)
    if code != 200:
        raise SystemExit(f"FAIL Grafana health HTTP {code}")
    print(f"OK  Grafana health ({GRAFANA_HEALTH})")

    send_smoke_span()
    assert_trace_searchable()

    print("—" * 40)
    print("Grafana dashboards:")
    print("  Distributed Trace View  http://127.0.0.1:3000/d/distributed-trace-view")
    print(f"    tempo_trace_id={SMOKE_TRACE_HEX}  (smoke test)")
    print(
        "  Explore span drill-down "
        f"http://127.0.0.1:3000/explore?left={{\"datasource\":\"tempo\",\"queries\":[{{\"queryType\":\"traceId\",\"query\":\"{SMOKE_TRACE_HEX}\"}}]}}"
    )
    print("    (若面板 spanFilters 不可见，在 Explore 瀑布图中点击 span 验证 traces→logs 联动)")
    print("  Trace Log Journey       http://127.0.0.1:3000/d/trace-log-journey")
    print("All checks passed.")
    return 0


if __name__ == "__main__":
    sys.exit(main())
