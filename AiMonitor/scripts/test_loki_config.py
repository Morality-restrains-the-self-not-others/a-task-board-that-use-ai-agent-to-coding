"""Validate Loki retention is configured for local dev (7 days)."""

from __future__ import annotations

from pathlib import Path

import yaml


def test_loki_retention_is_seven_days() -> None:
    path = Path(__file__).resolve().parents[1] / "loki" / "loki-config.yaml"
    data = yaml.safe_load(path.read_text(encoding="utf-8"))
    limits = data.get("limits_config") or {}
    assert limits.get("retention_period") == "168h"
    compactor = data.get("compactor") or {}
    assert compactor.get("retention_enabled") is True
    assert compactor.get("delete_request_store") == "filesystem"
    assert "delete_delay" not in compactor, "delete_delay removed for Loki 3.4.x compatibility"
