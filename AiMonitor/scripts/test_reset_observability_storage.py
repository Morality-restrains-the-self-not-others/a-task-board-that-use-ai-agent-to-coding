"""Validate reset_observability_storage.sh structure and ordering."""

from __future__ import annotations

from pathlib import Path


def test_reset_script_exists_and_has_stop_start_order() -> None:
    script = Path(__file__).resolve().parent / "reset_observability_storage.sh"
    assert script.is_file(), "reset_observability_storage.sh must exist"
    text = script.read_text(encoding="utf-8")
    assert "compose stop" in text
    assert "compose rm -f" in text
    assert "compose up -d --force-recreate" in text
    assert "prometheus tempo otel-collector loki" in text
    compose_up_line = next(
        (line for line in text.splitlines() if "compose up -d --force-recreate" in line),
        "",
    )
    assert "promtail" not in compose_up_line

    stop_idx = text.find('compose stop "${SERVICES[@]}"')
    compose_up = text.find("compose up -d --force-recreate")
    assert stop_idx >= 0 and compose_up > stop_idx

    for vol in ("loki_data", "tempo_data", "prometheus_data"):
        assert vol in text
    assert "runall-local-promtail.sh" in text

    assert "grafana_data" not in text or "Preserves grafana_data" in text


def test_reset_script_starts_promtail_after_loki_recreate() -> None:
    """Wiping Loki without `promtail up` leaves Grafana Trace Log Journey empty."""
    script = Path(__file__).resolve().parent / "reset_observability_storage.sh"
    text = script.read_text(encoding="utf-8")
    compose_up = text.find("compose up -d --force-recreate")
    up_call = text.find('"$LOCAL_PROMTAIL_RESET" up')
    assert up_call > compose_up, "must run runall-local-promtail.sh up after Loki recreate"
