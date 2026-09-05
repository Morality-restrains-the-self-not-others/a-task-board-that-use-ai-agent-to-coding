"""Validate Trace Log Explore Grafana dashboard filters."""

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
        / "trace-log-explore.json"
    )
    return json.loads(path.read_text(encoding="utf-8"))


def _variable_by_name(data: dict, name: str) -> dict | None:
    variables = data.get("templating", {}).get("list", [])
    return next((v for v in variables if v.get("name") == name), None)


def test_trace_log_explore_has_level_variable() -> None:
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


def test_trace_log_explore_has_search_variable() -> None:
    data = _load_dashboard()
    search_var = _variable_by_name(data, "search")
    assert search_var is not None, "search template variable required"
    assert search_var.get("type") == "textbox"
    assert search_var.get("label") == "search"


def test_trace_log_explore_has_msg_variable() -> None:
    data = _load_dashboard()
    msg_var = _variable_by_name(data, "msg")
    assert msg_var is not None, "msg template variable required"
    assert msg_var.get("type") == "query"
    assert msg_var.get("includeAll") is True
    assert msg_var.get("allValue") == ".*"
    assert "label_values(msg)" in str(msg_var.get("query", ""))


def test_trace_log_explore_has_adhoc_filters_variable() -> None:
    data = _load_dashboard()
    filters_var = _variable_by_name(data, "Filters")
    assert filters_var is not None, "Filters adhoc template variable required"
    assert filters_var.get("type") == "adhoc"
    assert filters_var.get("datasource", {}).get("uid") == "loki"


def test_trace_log_explore_logql_filters() -> None:
    data = _load_dashboard()
    exprs = [
        target.get("expr", "")
        for panel in data.get("panels") or []
        for target in panel.get("targets") or []
    ]
    assert exprs, "dashboard must have LogQL targets"
    expr = next((e for e in exprs if "$level" in e), "")
    assert expr, exprs
    assert 'level=~"(?i)($level)"' in expr
    assert 'service=~"$service"' in expr
    assert 'msg=~"$msg"' in expr
    assert '|= "$trace_id"' in expr
    assert '|= "$search"' in expr


def test_trace_log_explore_uid() -> None:
    data = _load_dashboard()
    assert data.get("uid") == "trace-log-explore"
