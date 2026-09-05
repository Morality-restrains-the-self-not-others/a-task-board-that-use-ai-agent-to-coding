"""Validate OTel Collector forwards traces to Tempo."""

from __future__ import annotations

from pathlib import Path

import yaml


def test_otel_collector_exports_traces_to_tempo() -> None:
    path = Path(__file__).resolve().parents[1] / "otel-collector" / "otel-collector.yaml"
    data = yaml.safe_load(path.read_text(encoding="utf-8"))
    assert isinstance(data, dict)

    receivers = data.get("receivers") or {}
    otlp = receivers.get("otlp") or {}
    protocols = otlp.get("protocols") or {}
    assert "grpc" in protocols
    assert "http" in protocols

    exporters = data.get("exporters") or {}
    tempo = exporters.get("otlp/tempo") or {}
    assert tempo.get("endpoint") == "tempo:4317"

    pipelines = (data.get("service") or {}).get("pipelines") or {}
    traces = pipelines.get("traces") or {}
    assert "otlp" in traces.get("receivers", [])
    assert "otlp/tempo" in traces.get("exporters", [])
