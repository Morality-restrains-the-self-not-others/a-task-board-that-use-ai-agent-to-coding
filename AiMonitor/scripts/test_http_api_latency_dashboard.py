"""Validate HTTP API Latency Grafana dashboard structure and PromQL."""

from __future__ import annotations

import json
from pathlib import Path


def _load_dashboard() -> dict:
    path = (
        Path(__file__).resolve().parents[1]
        / "grafana"
        / "provisioning"
        / "dashboards"
        / "files"
        / "http-api-latency.json"
    )
    return json.loads(path.read_text(encoding="utf-8"))


def _variable_by_name(data: dict, name: str) -> dict | None:
    variables = data.get("templating", {}).get("list", [])
    return next((v for v in variables if v.get("name") == name), None)


def _panel_by_title(data: dict, title: str) -> dict | None:
    return next((p for p in data.get("panels") or [] if p.get("title") == title), None)


def _all_target_exprs(data: dict) -> list[str]:
    return [
        t.get("expr", "")
        for panel in data.get("panels") or []
        if panel.get("type") != "row"
        for t in (panel.get("targets") or [])
    ]


def test_http_api_latency_uid_and_title() -> None:
    data = _load_dashboard()
    assert data["uid"] == "http-api-latency"
    assert data["title"] == "HTTP API Latency"
    assert data["schemaVersion"] >= 39
    tags = data.get("tags") or []
    assert "aimonitor" in tags
    assert "latency" in tags
    assert "http" in tags


def test_http_api_latency_datasource_variable() -> None:
    data = _load_dashboard()
    ds = _variable_by_name(data, "DS_PROMETHEUS")
    assert ds is not None
    assert ds.get("type") == "datasource"
    assert ds.get("query") == "prometheus"


def test_http_api_latency_has_service_and_path_variables() -> None:
    data = _load_dashboard()
    svc = _variable_by_name(data, "service")
    assert svc is not None
    assert svc.get("type") == "query"
    assert svc.get("includeAll") is True
    assert "http_request_duration_seconds" in str(svc.get("query", ""))
    path = _variable_by_name(data, "path")
    assert path is not None
    assert path.get("includeAll") is True


def test_http_api_latency_has_min_latency_filter_variable() -> None:
    """Users must be able to hide series slower than a duration threshold."""
    data = _load_dashboard()
    var = _variable_by_name(data, "min_latency")
    assert var is not None
    assert var.get("type") == "custom"
    assert var.get("multi") is False
    query = str(var.get("query", ""))
    # Grafana 11 puts the custom option into the URL; values must be PromQL-safe
    # durations (100ms, 1s) so `$min_latency` interpolates as `> 100ms`, not a label.
    assert "100ms" in query
    assert "500ms" in query
    assert "1s" in query
    assert " : " not in query
    assert var.get("includeAll") is True
    assert str(var.get("allValue")) == "0"


def test_http_api_latency_overview_stats() -> None:
    data = _load_dashboard()
    for title in ("Request Rate", "Error Rate", "P95 Latency", "P99 Latency"):
        panel = _panel_by_title(data, title)
        assert panel is not None, f"missing stat panel {title}"
        assert panel.get("type") == "stat"


def test_http_api_latency_percentile_timeseries() -> None:
    data = _load_dashboard()
    panel = _panel_by_title(data, "Latency percentiles (p50/p95/p99)")
    assert panel is not None
    assert panel.get("type") == "timeseries"
    exprs = [t.get("expr", "") for t in panel.get("targets") or []]
    assert len(exprs) == 3
    blob = "\n".join(exprs)
    assert "histogram_quantile(0.5" in blob
    assert "histogram_quantile(0.95" in blob
    assert "histogram_quantile(0.99" in blob
    assert "http_request_duration_seconds_bucket" in blob


def test_http_api_latency_has_service_and_path_p95() -> None:
    data = _load_dashboard()
    by_svc = _panel_by_title(data, "P95 by service")
    assert by_svc is not None
    assert by_svc.get("type") == "timeseries"
    expr = (by_svc.get("targets") or [{}])[0].get("expr", "")
    assert "histogram_quantile(0.95" in expr
    assert "service" in expr
    assert "> $min_latency" in expr, expr
    by_path = _panel_by_title(data, "P95 by path")
    assert by_path is not None
    expr = (by_path.get("targets") or [{}])[0].get("expr", "")
    assert "path" in expr
    assert "> $min_latency" in expr, expr
    gw = _panel_by_title(data, "Gateway (APISIX) p95 by route")
    assert gw is not None
    gw_expr = (gw.get("targets") or [{}])[0].get("expr", "")
    assert "> $min_latency" in gw_expr, gw_expr


def test_http_api_latency_slow_paths_table_is_filterable() -> None:
    data = _load_dashboard()
    panel = _panel_by_title(data, "Paths with p95 ≥ min latency")
    assert panel is not None
    assert panel.get("type") == "table"
    target = (panel.get("targets") or [{}])[0]
    expr = target.get("expr", "")
    assert "histogram_quantile(0.95" in expr
    assert "> $min_latency" in expr
    assert "$__range" in expr
    assert target.get("instant") is True
    assert target.get("format") == "table"
    filterable = (
        (panel.get("fieldConfig") or {})
        .get("defaults", {})
        .get("custom", {})
        .get("filterable")
    )
    assert filterable is True


def test_http_api_latency_slow_path_count_uses_min_latency() -> None:
    data = _load_dashboard()
    panel = _panel_by_title(data, "Slow path count")
    assert panel is not None
    assert panel.get("type") == "stat"
    expr = (panel.get("targets") or [{}])[0].get("expr", "")
    assert "histogram_quantile(0.95" in expr
    assert "> $min_latency" in expr
    assert "count(" in expr


def test_http_api_latency_has_red_traffic_panels() -> None:
    data = _load_dashboard()
    rate = _panel_by_title(data, "Request rate by service")
    assert rate is not None
    expr = (rate.get("targets") or [{}])[0].get("expr", "")
    assert "http_requests_total" in expr
    err = _panel_by_title(data, "Error rate by service")
    assert err is not None
    expr = (err.get("targets") or [{}])[0].get("expr", "")
    assert "5.." in expr or 'status=~"5' in expr


def test_http_api_latency_queries_use_rate_interval_and_filters() -> None:
    data = _load_dashboard()
    exprs = _all_target_exprs(data)
    assert exprs
    for expr in exprs:
        if "rate(" in expr:
            assert "$__rate_interval" in expr or "$__range" in expr, expr
        if "apisix_http_latency" in expr:
            continue
        assert "$service" in expr, expr
        assert "$path" in expr, expr
        assert "{__name__=~" in expr or "__name__=~" in expr, expr


def test_http_api_latency_listed_on_dashboards_uid() -> None:
    """Dashboards page discovers provisioned files by uid; keep it stable."""
    data = _load_dashboard()
    assert data["uid"] == "http-api-latency"
    assert "editable" in data
