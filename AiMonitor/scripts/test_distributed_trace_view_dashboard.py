"""Validate Distributed Trace View Grafana dashboard Tempo/Loki span linkage."""

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
        / "distributed-trace-view.json"
    )
    return json.loads(path.read_text(encoding="utf-8"))


def _variable_by_name(data: dict, name: str) -> dict | None:
    variables = data.get("templating", {}).get("list", [])
    return next((v for v in variables if v.get("name") == name), None)


def test_distributed_trace_view_has_span_id_variable() -> None:
    data = _load_dashboard()
    span_var = _variable_by_name(data, "span_id")
    assert span_var is not None, "span_id template variable required"
    assert span_var.get("type") == "textbox"


def test_distributed_trace_view_has_loki_span_hierarchy_panel() -> None:
    data = _load_dashboard()
    titles = [p.get("title", "") for p in data.get("panels") or []]
    assert any("Loki span hierarchy" in t for t in titles), titles


def test_distributed_trace_view_tempo_panel_uses_trace_id_variable() -> None:
    data = _load_dashboard()
    tempo_panel = next(p for p in data.get("panels") or [] if p.get("type") == "traces")
    assert tempo_panel["targets"][0]["query"] == "$tempo_trace_id"
    assert "span tree" in tempo_panel.get("title", "").lower()


def test_distributed_trace_view_links_pass_span_variables() -> None:
    data = _load_dashboard()
    links = data.get("links") or []
    assert any("var-span_id=${span_id}" in link.get("url", "") for link in links)


def test_distributed_trace_view_has_explore_span_drilldown_link() -> None:
    data = _load_dashboard()
    links = data.get("links") or []
    explore = next((link for link in links if link.get("title") == "Explore span drill-down"), None)
    assert explore is not None, links
    assert "traceId" in explore.get("url", "")
    assert "${tempo_trace_id}" in explore.get("url", "")


def test_distributed_trace_view_tempo_panel_documents_spanfilter_fallback() -> None:
    data = _load_dashboard()
    tempo_panel = next(p for p in data.get("panels") or [] if p.get("type") == "traces")
    desc = tempo_panel.get("description", "")
    assert "Explore" in desc or "explore" in desc.lower()
    assert tempo_panel.get("options", {}).get("showSpanFilters") is True


def test_distributed_trace_view_skips_json_parser_errors() -> None:
    data = _load_dashboard()
    exprs = [
        target.get("expr", "")
        for panel in data.get("panels") or []
        for target in panel.get("targets") or []
        if "| json" in target.get("expr", "")
    ]
    assert exprs, "expected Loki | json queries"
    for expr in exprs:
        assert '| __error__=""' in expr, expr
