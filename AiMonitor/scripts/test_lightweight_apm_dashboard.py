"""Validate Lightweight APM Grafana dashboard structure and queries."""

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
        / "lightweight-apm.json"
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


# --- Metadata ---

def test_lightweight_apm_uid_unique() -> None:
    data = _load_dashboard()
    assert data["uid"] == "lightweight-apm"


def test_lightweight_apm_schema_version() -> None:
    data = _load_dashboard()
    assert data["schemaVersion"] >= 39


def test_lightweight_apm_tags() -> None:
    data = _load_dashboard()
    tags = data.get("tags") or []
    assert "apm" in tags
    assert "aimonitor" in tags


# --- Template Variables ---

def test_lightweight_apm_has_service_variable() -> None:
    data = _load_dashboard()
    svc = _variable_by_name(data, "service")
    assert svc is not None, "service template variable required"
    assert svc.get("type") == "query"
    assert svc.get("includeAll") is True
    query = str(svc.get("query", ""))
    assert "traces_spanmetrics_calls_total" in query
    assert "service_name" not in query
    assert "label_values(traces_spanmetrics_calls_total, service)" in query


def test_lightweight_apm_uses_tempo_metrics_generator_names() -> None:
    """Tempo 2.x metrics-generator emits latency_* and label service, not OTEL names."""
    data = _load_dashboard()
    exprs = _all_target_exprs(data)
    span_exprs = [e for e in exprs if "traces_spanmetrics_" in e]
    assert span_exprs, "expected spanmetrics PromQL"
    joined = "\n".join(span_exprs)
    assert "traces_spanmetrics_duration_seconds" not in joined
    assert "traces_spanmetrics_latency_bucket" in joined
    assert "service_name" not in joined
    assert "SPAN_KIND_SERVER" not in joined
    assert 'service=~"$service"' in joined


def test_lightweight_apm_has_loki_service_variable() -> None:
    data = _load_dashboard()
    svc = _variable_by_name(data, "loki_service")
    assert svc is not None, "loki_service template variable required"
    assert svc.get("type") == "query"
    assert "label_values(job)" in str(svc.get("query", ""))


def test_lightweight_apm_has_trace_id_variables() -> None:
    data = _load_dashboard()
    for name in ("trace_id", "tempo_trace_id"):
        var = _variable_by_name(data, name)
        assert var is not None, f"{name} template variable required"
        assert var.get("type") == "textbox"


# --- Panels: Overview ---

def test_lightweight_apm_has_overview_stats() -> None:
    data = _load_dashboard()
    for title in ("Active Services", "Total Request Rate", "Overall Error Rate", "P99 Latency (Overall)"):
        panel = _panel_by_title(data, title)
        assert panel is not None, f"Missing overview stat panel: {title}"
        assert panel.get("type") == "stat", f"{title} must be stat panel"


# --- Panels: RED ---

def test_lightweight_apm_has_red_panels() -> None:
    data = _load_dashboard()
    for title in ("Request Rate by Service", "Error Rate % by Service", "P99 Latency by Service"):
        panel = _panel_by_title(data, title)
        assert panel is not None, f"Missing RED panel: {title}"
        assert panel.get("type") == "timeseries", f"{title} must be timeseries"


# --- Panels: Latency Analysis ---

def test_lightweight_apm_has_latency_heatmap() -> None:
    data = _load_dashboard()
    panel = _panel_by_title(data, "Latency Distribution Heatmap")
    assert panel is not None, "Missing latency heatmap"
    assert panel.get("type") == "heatmap"


def test_lightweight_apm_has_percentile_panel() -> None:
    data = _load_dashboard()
    panel = _panel_by_title(data, "Latency Percentiles (p50/p90/p99)")
    assert panel is not None, "Missing latency percentiles panel"
    targets = panel.get("targets") or []
    # Must have p50, p90, p99 queries
    assert len(targets) == 3, f"Expected 3 targets (p50/p90/p99), got {len(targets)}"


# --- Panels: Dependencies ---

def test_lightweight_apm_has_service_dependency_panels() -> None:
    data = _load_dashboard()
    for title in ("Service-to-Service Call Rate", "Service Dependency Table", "Service-to-Service P99 Latency"):
        panel = _panel_by_title(data, title)
        assert panel is not None, f"Missing dependency panel: {title}"


def test_lightweight_apm_has_status_code_distribution() -> None:
    data = _load_dashboard()
    panel = _panel_by_title(data, "HTTP Status Code Distribution")
    assert panel is not None, "Missing HTTP status distribution panel"


# --- Panels: Error Analysis (Loki) ---

def test_lightweight_apm_has_error_log_panels() -> None:
    data = _load_dashboard()
    for title in ("Error Log Volume by Service", "Recent Errors (table)", "Error Log Stream"):
        panel = _panel_by_title(data, title)
        assert panel is not None, f"Missing error log panel: {title}"


def test_lightweight_apm_error_log_panels_use_loki() -> None:
    data = _load_dashboard()
    for title in ("Error Log Volume by Service", "Recent Errors (table)", "Error Log Stream"):
        panel = _panel_by_title(data, title)
        if panel is None:
            continue
        ds = panel.get("datasource", {})
        assert ds.get("type") == "loki", f"{title} must use Loki datasource"


def test_lightweight_apm_error_logs_filter_level() -> None:
    """Error log panels must filter by error/fatal/panic level."""
    data = _load_dashboard()
    exprs = _all_target_exprs(data)
    loki_exprs = [e for e in exprs if "{job=" in e and ("error" in e.lower())]
    for expr in loki_exprs:
        assert "extracted_level" in expr, f"Loki error query missing regexp extraction: {expr[:80]}..."
        assert "level=~" in expr, f"Loki error query missing level filter: {expr[:80]}..."
        assert '| __error__=""' in expr, f"Loki error query must skip JSONParserErr: {expr[:80]}..."


# --- Panels: Health ---

def test_lightweight_apm_has_health_score() -> None:
    data = _load_dashboard()
    panel = _panel_by_title(data, "Service Health Score")
    assert panel is not None, "Missing health score panel"
    target_exprs = [t.get("expr", "") for t in panel.get("targets") or []]
    assert any("probe_success" in e for e in target_exprs), "Health score must use probe_success"


# --- Cross-links ---

def test_lightweight_apm_has_cross_dashboard_links() -> None:
    data = _load_dashboard()
    links = data.get("links") or []
    link_titles = [l.get("title", "") for l in links]
    assert any("Distributed Trace" in t for t in link_titles), "Missing link to Distributed Trace View"
    assert any("Trace Log Journey" in t for t in link_titles), "Missing link to Trace Log Journey"


# --- Query validation ---

def test_lightweight_apm_uses_rate_interval() -> None:
    """All Prometheus rate queries must use $__rate_interval for Grafana >= 10."""
    data = _load_dashboard()
    exprs = _all_target_exprs(data)
    rate_exprs = [e for e in exprs if "rate(" in e]
    for expr in rate_exprs:
        assert "$__rate_interval" in expr, (
            f"rate query must use $__rate_interval, not hardcoded window: {expr[:80]}..."
        )


def test_lightweight_apm_queries_reference_service_variable() -> None:
    """Span metric queries in RED/dependency panels must filter by $service variable.

    Overview stat panels (Active Services, Total Request Rate, etc.) are allowed
    to omit the filter because they intentionally show global aggregates.
    """
    data = _load_dashboard()
    # Exclude overview stat panels which are meant to be global
    overview_titles = {"Active Services", "Total Request Rate", "Overall Error Rate", "P99 Latency (Overall)",
                       "All Services Request Rate"}
    exprs = [
        t.get("expr", "")
        for panel in data.get("panels") or []
        if panel.get("type") != "row" and panel.get("title") not in overview_titles
        for t in (panel.get("targets") or [])
    ]
    span_exprs = [e for e in exprs if "traces_spanmetrics_" in e or "traces_service_graph_" in e]
    for expr in span_exprs:
        assert "$service" in expr, (
            f"Span metric query must filter by $service variable: {expr[:80]}..."
        )


def test_lightweight_apm_loki_queries_use_loki_service() -> None:
    """All Loki log-stream queries must filter by $loki_service variable.

    Only matches Loki log stream selectors (pattern: {job=~"..."). PromQL queries
    like probe_success{job="runall-health"} are not Loki queries and are excluded.
    """
    data = _load_dashboard()
    exprs = _all_target_exprs(data)
    # Loki log stream selectors start with {job= — not PromQL metric{job= filters
    loki_exprs = [
        e for e in exprs
        if e.strip().startswith("{job=") and ("|" in e or "|~" in e or "json" in e)
    ]
    for expr in loki_exprs:
        assert "$loki_service" in expr, (
            f"Loki query must filter by $loki_service variable: {expr[:80]}..."
        )
