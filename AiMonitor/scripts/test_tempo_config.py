"""Validate AiMonitor Tempo config for local trace storage."""

from __future__ import annotations

from pathlib import Path

import yaml


def test_tempo_has_otlp_receiver_and_retention() -> None:
    path = Path(__file__).resolve().parents[1] / "tempo" / "tempo-config.yaml"
    data = yaml.safe_load(path.read_text(encoding="utf-8"))
    assert isinstance(data, dict)

    distributor = data.get("distributor") or {}
    receivers = distributor.get("receivers") or {}
    otlp = receivers.get("otlp") or {}
    protocols = otlp.get("protocols") or {}
    assert "grpc" in protocols
    assert "http" in protocols

    compactor = data.get("compactor") or {}
    compaction = compactor.get("compaction") or {}
    assert compaction.get("block_retention") == "168h"

    storage = data.get("storage") or {}
    trace = storage.get("trace") or {}
    assert trace.get("backend") == "local"

    metrics = data.get("metrics_generator") or {}
    storage = metrics.get("storage") or {}
    remote = storage.get("remote_write") or []
    assert any("prometheus:9090" in str(r.get("url", "")) for r in remote)
    processors = (data.get("overrides") or {}).get("defaults", {}).get("metrics_generator", {}).get("processors") or []
    assert "service-graphs" in processors

    server = data.get("server") or {}
    assert server.get("http_listen_port") == 3200
