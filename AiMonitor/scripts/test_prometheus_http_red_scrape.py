"""Validate Prometheus scrape of HTTP RED /metrics endpoints."""

from __future__ import annotations

from pathlib import Path

import yaml


def _load_prometheus() -> dict:
    path = Path(__file__).resolve().parents[1] / "prometheus" / "prometheus.yml"
    return yaml.safe_load(path.read_text(encoding="utf-8"))


def _job(data: dict, name: str) -> dict:
    jobs = data.get("scrape_configs") or []
    job = next((j for j in jobs if j.get("job_name") == name), None)
    assert job is not None, f"missing scrape job {name}"
    return job


def _service_labels(job: dict) -> dict[str, str]:
    out: dict[str, str] = {}
    for cfg in job.get("static_configs") or []:
        labels = cfg.get("labels") or {}
        service = labels.get("service")
        for target in cfg.get("targets") or []:
            assert service, f"target {target} in job {job.get('job_name')} missing service label"
            out[str(target)] = str(service)
    return out


def test_taskauth_metrics_has_service_label() -> None:
    job = _job(_load_prometheus(), "taskauth-metrics")
    labels = _service_labels(job)
    assert labels.get("host.docker.internal:8003") == "task-auth"


def test_task_bill_metrics_scraped() -> None:
    job = _job(_load_prometheus(), "task-bill-metrics")
    assert job.get("metrics_path") in (None, "/metrics")
    labels = _service_labels(job)
    assert labels.get("host.docker.internal:8004") == "task-bill"


def test_v6_go_services_labeled_per_instance() -> None:
    job = _job(_load_prometheus(), "v6-go-services")
    assert job.get("metrics_path") == "/api/metrics"
    labels = _service_labels(job)
    assert labels.get("host.docker.internal:8016") == "task-project-service"
    assert labels.get("host.docker.internal:8017") == "task-task-service"
    assert labels.get("host.docker.internal:8018") == "task-cloud-service"


def test_http_red_jobs_keep_stable_names_for_alerts() -> None:
    data = _load_prometheus()
    names = {j.get("job_name") for j in data.get("scrape_configs") or []}
    assert "taskauth-metrics" in names
    assert "v6-go-services" in names
