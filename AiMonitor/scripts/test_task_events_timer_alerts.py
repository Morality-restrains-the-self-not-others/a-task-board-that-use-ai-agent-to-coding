"""Validate task-events timer liveness alert rules (OPT-20260827-049)."""

from __future__ import annotations

import json
from pathlib import Path

import yaml

TIMER_JOB = "task-events-queued-auto-run-scan-1-dispatch"


def _rules_path() -> Path:
    return (
        Path(__file__).resolve().parents[1]
        / "prometheus"
        / "rules"
        / "task-events-timer-alerts.yml"
    )


def _prometheus_yml_path() -> Path:
    return Path(__file__).resolve().parents[1] / "prometheus" / "prometheus.yml"


def _file_sd_path() -> Path:
    return (
        Path(__file__).resolve().parents[1]
        / "prometheus"
        / "file_sd"
        / "runall-health-targets.json"
    )


def test_queued_auto_run_scan_down_alert_defined() -> None:
    data = yaml.safe_load(_rules_path().read_text(encoding="utf-8"))
    rules = data["groups"][0]["rules"]
    alert = next(r for r in rules if r["alert"] == "TaskEventsQueuedAutoRunScanDown")
    expr = alert["expr"]
    assert f'up{{job="{TIMER_JOB}"}} == 0' in expr, expr
    assert alert["for"] in ("3m", "5m", "10m"), "alert needs a 'for' window"


def test_timer_alert_job_exists_in_file_sd() -> None:
    """The alert's job label must match a Prometheus file_sd health target."""
    payload = json.loads(_file_sd_path().read_text(encoding="utf-8"))
    jobs = {
        t["labels"]["service"]
        for t in payload
        if "service" in t.get("labels", {})
    }
    assert TIMER_JOB in jobs, f"missing file_sd health target for {TIMER_JOB}"


def test_timer_alert_rule_file_registered() -> None:
    data = yaml.safe_load(_prometheus_yml_path().read_text(encoding="utf-8"))
    assert "/etc/prometheus/rules/task-events-timer-alerts.yml" in data["rule_files"]
