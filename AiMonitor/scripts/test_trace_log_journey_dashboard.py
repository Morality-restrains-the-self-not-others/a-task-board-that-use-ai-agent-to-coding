"""Validate Trace Log Journey Grafana dashboard trace correlation queries."""

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
        / "trace-log-journey.json"
    )
    return json.loads(path.read_text(encoding="utf-8"))


def _variable_by_name(data: dict, name: str) -> dict | None:
    variables = data.get("templating", {}).get("list", [])
    return next((v for v in variables if v.get("name") == name), None)


def test_trace_log_journey_has_service_variable() -> None:
    data = _load_dashboard()
    service_var = _variable_by_name(data, "service")
    assert service_var is not None, "service template variable required to narrow job scope"
    assert service_var.get("includeAll") is True
    assert service_var.get("allValue") in (".+", ".*")
    assert "label_values(job)" in str(service_var.get("query", ""))


def test_trace_log_journey_has_tempo_trace_id_variable() -> None:
    data = _load_dashboard()
    tempo_var = _variable_by_name(data, "tempo_trace_id")
    assert tempo_var is not None, "tempo_trace_id template variable required"
    assert tempo_var.get("type") == "textbox"


def test_trace_log_journey_has_span_id_variable() -> None:
    data = _load_dashboard()
    span_var = _variable_by_name(data, "span_id")
    assert span_var is not None, "span_id template variable required"


def test_trace_log_journey_has_level_variable() -> None:
    data = _load_dashboard()
    level_var = _variable_by_name(data, "level")
    assert level_var is not None, "level template variable required"
    assert level_var.get("type") == "custom"
    assert level_var.get("includeAll") is True
    assert level_var.get("allValue") == ".*"
    assert level_var.get("multi") is True
    query = str(level_var.get("query", ""))
    assert "debug" in query and "info" in query and "error" in query
    assert "warn|warning" in query


def test_trace_log_journey_has_span_hierarchy_panel() -> None:
    data = _load_dashboard()
    titles = [p.get("title", "") for p in data.get("panels") or []]
    assert any("Span hierarchy" in t for t in titles), titles


def test_trace_log_journey_logql_matches_trace_and_otel() -> None:
    data = _load_dashboard()
    exprs = [
        target.get("expr", "")
        for panel in data.get("panels") or []
        for target in panel.get("targets") or []
    ]
    assert any('|~ "$trace_id|$tempo_trace_id"' in expr for expr in exprs), exprs


def test_trace_log_journey_logql_filters_by_level_case_insensitive() -> None:
    data = _load_dashboard()
    exprs = [
        target.get("expr", "")
        for panel in data.get("panels") or []
        for target in panel.get("targets") or []
    ]
    assert exprs, "dashboard must have LogQL targets"
    assert all('level=~"(?i)($level)"' in expr for expr in exprs), exprs
    # Plaintext startup errors lack the level label; coalesce via regexp before filtering.
    assert all("| regexp `" in expr for expr in exprs), exprs
    assert all("| label_format level=" in expr for expr in exprs), exprs
    assert all('{job=~".+", level=~"(?i)($level)"}' not in expr for expr in exprs), exprs
    assert all('job=~"$service"' in expr for expr in exprs), exprs


def test_trace_log_journey_has_status_variable() -> None:
    data = _load_dashboard()
    status_var = _variable_by_name(data, "status")
    assert status_var is not None, "status template variable required"
    assert status_var.get("type") == "custom"
    assert status_var.get("includeAll") is True
    assert status_var.get("allValue") == ".*"
    query = str(status_var.get("query", ""))
    assert "404" in query and "5xx" in query


def test_trace_log_journey_status_values_are_field_level() -> None:
    """Status filter values must be field-level patterns, not full-line regex.

    After JSON parsing, status=~ filters the extracted status label.
    Full-line patterns like '"status"\\s*:\\s*404\\b' would not work correctly
    with label filtering and also over-match on non-status fields.
    """
    data = _load_dashboard()
    status_var = _variable_by_name(data, "status")
    assert status_var is not None
    values = [opt.get("value", "") for opt in status_var.get("options", [])]
    # No value should contain the old full-line regex pattern
    for v in values:
        assert '\\s*' not in v, f"status value uses old full-line regex: {v!r}"
        assert '\\b' not in v, f"status value uses old full-line regex: {v!r}"
        assert '"status"' not in v, f"status value has hardcoded JSON key: {v!r}"


def test_trace_log_journey_status_filter_uses_json_first() -> None:
    """Status filter must parse JSON before matching status label.

    Old approach: |~ `($status)` — regex on raw line, matches non-status fields.
    New approach: | json | label_format status=... | status=~"($status)" — precise.
    """
    data = _load_dashboard()
    exprs = [
        target.get("expr", "")
        for panel in data.get("panels") or []
        for target in panel.get("targets") or []
    ]
    # Verify no query still uses old full-line regex filter for status
    for expr in exprs:
        assert '|~ `($status)`' not in expr, (
            f"Query still uses old full-line status filter: ...{expr[-100:]}"
        )
    # Verify all queries use field-level status filter after JSON parsing
    for expr in exprs:
        assert '| json' in expr, (
            f"Query missing | json before status filter: ...{expr[-100:]}"
        )
        assert '| label_format status=' in expr, (
            f"Query missing label_format for status default: ...{expr[-100:]}"
        )
        assert 'status=~"($status)"' in expr, (
            f"Query missing field-level status filter: ...{expr[-100:]}"
        )


def test_trace_log_journey_skips_json_parser_errors() -> None:
    """Metric + log panels must skip truncated/non-JSON lines.

    Loki count_over_time aborts the whole query on JSONParserErr (truncated
    slog lines, runAll prefix leftovers). | json | __error__="" keeps volume
    and stream panels rendering when most lines are valid JSON.
    """
    data = _load_dashboard()
    exprs = [
        target.get("expr", "")
        for panel in data.get("panels") or []
        for target in panel.get("targets") or []
    ]
    assert exprs, "dashboard must have LogQL targets"
    for expr in exprs:
        json_idx = expr.find("| json")
        assert json_idx >= 0, expr
        after = expr[json_idx:]
        assert '| __error__=""' in after or "| __error__=``" in after, (
            f"Query missing JSONParserErr skip after | json: ...{expr[-120:]}"
        )
