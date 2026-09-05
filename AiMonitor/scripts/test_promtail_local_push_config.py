"""Validate local Promtail pushes to remote Loki, not in-cluster loki hostname."""

from __future__ import annotations

from pathlib import Path

import yaml


def test_promtail_local_push_url_uses_env_not_docker_loki() -> None:
    path = Path(__file__).resolve().parents[1] / "promtail" / "promtail-local.yaml"
    data = yaml.safe_load(path.read_text(encoding="utf-8"))
    clients = data.get("clients") or []
    assert clients, "clients required"
    url = str(clients[0].get("url", ""))
    assert "${LOKI_PUSH_URL}" in url
    assert "http://loki:" not in url


def test_promtail_local_covers_all_runall_services() -> None:
    """Every runAll service must have a scrape_config so trace shipping can verify Loki."""
    root = Path(__file__).resolve().parents[2]
    runall_yaml = root / "conf" / "runAll.yaml"
    promtail_yaml = Path(__file__).resolve().parents[1] / "promtail" / "promtail-local.yaml"

    runall = yaml.safe_load(runall_yaml.read_text(encoding="utf-8"))
    promtail = yaml.safe_load(promtail_yaml.read_text(encoding="utf-8"))

    expected = {
        str(s.get("name", "")).strip()
        for g in runall.get("groups", [])
        for s in g.get("services", [])
        if str(s.get("name", "")).strip()
    }
    configured = {str(j.get("job_name", "")).strip() for j in promtail.get("scrape_configs", [])}

    missing = sorted(expected - configured)
    assert not missing, f"promtail-local.yaml missing scrape_configs for: {', '.join(missing)}"


def test_promtail_pipeline_accepts_raw_json_and_runall_prefix() -> None:
    """Pipeline must parse both runAll-prefixed lines and raw slog JSON."""
    import re

    path = Path(__file__).resolve().parents[1] / "promtail" / "promtail-local.yaml"
    data = yaml.safe_load(path.read_text(encoding="utf-8"))
    jobs = data.get("scrape_configs") or []
    assert jobs, "scrape_configs required"
    stages = jobs[0].get("pipeline_stages") or []
    regex_stage = next((s for s in stages if "regex" in s and "payload" in str(s)), None)
    assert regex_stage is not None
    expr = regex_stage["regex"]["expression"]
    assert "(?:" in expr or expr.startswith("^("), f"expected optional prefix group, got {expr}"
    pat = re.compile(expr)
    prefixed = '2026-07-10T10:00:00Z (stdout) {"msg":"ok","service":"task-cloud-service","trace_id":"web-abc12345"}'
    raw = '{"msg":"ok","service":"task-cloud-service","trace_id":"web-abc12345"}'
    m1 = pat.match(prefixed)
    m2 = pat.match(raw)
    assert m1 and m1.group("payload").startswith("{")
    assert m2 and m2.group("payload").startswith("{")


def test_promtail_pipeline_extracts_level_from_plaintext_errors() -> None:
    """Non-JSON error lines (log.Fatalf) must still receive level=error for Grafana filters."""
    path = Path(__file__).resolve().parents[1] / "promtail" / "promtail-local.yaml"
    data = yaml.safe_load(path.read_text(encoding="utf-8"))
    stages = (data.get("scrape_configs") or [{}])[0].get("pipeline_stages") or []
    plain = next(
        (
            s
            for s in stages
            if "regex" in s and "plain_level" in str(s.get("regex", {}).get("expression", ""))
        ),
        None,
    )
    assert plain is not None, "plain_level regex stage required for plaintext errors"
    level_tmpl = next(
        (
            s
            for s in stages
            if "template" in s
            and s["template"].get("source") == "level"
            and "plain_level" in str(s["template"].get("template", ""))
        ),
        None,
    )
    assert level_tmpl is not None, "level template must coalesce JSON level with plain_level"


def test_promtail_pipeline_has_timestamp_stage_from_ts() -> None:
    """OPT-20260831-021: pipeline 须用 JSON ts 作为 Loki 时间戳（RFC3339Nano）。"""
    path = Path(__file__).resolve().parents[1] / "promtail" / "promtail-local.yaml"
    data = yaml.safe_load(path.read_text(encoding="utf-8"))
    jobs = data.get("scrape_configs") or []
    assert jobs, "scrape_configs required"
    stages = jobs[0].get("pipeline_stages") or []
    ts_stage = next((s for s in stages if "timestamp" in s), None)
    assert ts_stage is not None, "timestamp stage required (source=ts, format=RFC3339Nano)"
    assert ts_stage["timestamp"]["source"] == "ts"
    assert ts_stage["timestamp"]["format"] == "RFC3339Nano"


def test_promtail_local_health_probes_container_not_loki() -> None:
    """Loki /ready is not Promtail. Missing container + ready Loki → empty Grafana."""
    root = Path(__file__).resolve().parents[2]
    runall = yaml.safe_load((root / "conf" / "runAll.yaml").read_text(encoding="utf-8"))
    svc = next(
        s
        for g in runall.get("groups", [])
        for s in g.get("services", [])
        if s.get("name") == "promtail-local"
    )
    hc = svc.get("health_check") or {}
    exec_cmd = str(hc.get("exec") or "")
    assert "aimonitor-promtail" in exec_cmd
    assert "3100/ready" not in str(hc.get("url") or "")
