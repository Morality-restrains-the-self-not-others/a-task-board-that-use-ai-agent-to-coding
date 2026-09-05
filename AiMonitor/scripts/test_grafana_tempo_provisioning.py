"""Validate Grafana provisioning includes Tempo datasource with Loki correlation."""

from __future__ import annotations

import json
from pathlib import Path

import yaml


def test_grafana_tempo_datasource_provisioned() -> None:
    path = Path(__file__).resolve().parents[1] / "grafana" / "provisioning" / "datasources" / "prometheus.yaml"
    data = yaml.safe_load(path.read_text(encoding="utf-8"))
    datasources = data.get("datasources") or []
    tempo = next((d for d in datasources if d.get("type") == "tempo"), None)
    assert tempo is not None, "Tempo datasource required"
    assert tempo.get("uid") == "tempo"
    assert "tempo:3200" in tempo.get("url", "")

    json_data = tempo.get("jsonData") or {}
    traces_to_logs = json_data.get("tracesToLogsV2") or {}
    assert traces_to_logs.get("datasourceUid") == "loki"


def test_distributed_trace_dashboard_exists() -> None:
    path = (
        Path(__file__).resolve().parents[1]
        / "grafana"
        / "provisioning"
        / "dashboards"
        / "files"
        / "distributed-trace-view.json"
    )
    data = json.loads(path.read_text(encoding="utf-8"))
    assert data.get("uid") == "distributed-trace-view"
    panels = data.get("panels") or []
    types = {p.get("type") for p in panels}
    assert "traces" in types
    assert "logs" in types
    assert "nodeGraph" in types
    tempo_panel = next(p for p in panels if p.get("type") == "traces")
    assert tempo_panel["targets"][0]["query"] == "$tempo_trace_id"
    vars = {v["name"] for v in data.get("templating", {}).get("list", [])}
    assert "tempo_trace_id" in vars
    span_var = next(v for v in data.get("templating", {}).get("list", []) if v["name"] == "span_id")
    assert span_var.get("type") == "textbox"
    titles = [p.get("title", "") for p in panels]
    assert any("Loki span hierarchy" in t for t in titles)
