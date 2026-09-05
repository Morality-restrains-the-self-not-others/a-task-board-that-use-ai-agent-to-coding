"""Validate Service URL / API Catalog Grafana dashboard structure."""

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
        / "service-url-api-catalog.json"
    )
    return json.loads(path.read_text(encoding="utf-8"))


def _panel_by_title(data: dict, title: str) -> dict | None:
    return next((p for p in data.get("panels") or [] if p.get("title") == title), None)


def test_catalog_uid_title_tags() -> None:
    data = _load_dashboard()
    assert data["uid"] == "service-url-api-catalog"
    assert data["title"] == "Service URL / API Catalog"
    assert data["schemaVersion"] >= 39
    tags = data.get("tags") or []
    assert "aimonitor" in tags
    assert "catalog" in tags
    assert "core-path" in tags


def test_catalog_has_tempo_service_map() -> None:
    panel = _panel_by_title(_load_dashboard(), "Runtime service map (Tempo)")
    assert panel is not None
    assert panel.get("type") == "nodeGraph"
    targets = panel.get("targets") or []
    assert targets
    assert targets[0].get("queryType") == "serviceMap"


def test_catalog_has_prometheus_hot_paths() -> None:
    panel = _panel_by_title(_load_dashboard(), "Empirical hot paths (HTTP rate by path)")
    assert panel is not None
    exprs = [t.get("expr", "") for t in (panel.get("targets") or [])]
    assert any("http_requests_total" in e for e in exprs)


def test_catalog_decision_text_mentions_ssot() -> None:
    panel = _panel_by_title(_load_dashboard(), "Catalog decision")
    assert panel is not None
    content = (panel.get("options") or {}).get("content") or ""
    assert "SSOT" in content
    assert "298" in content or "API routes" in content
