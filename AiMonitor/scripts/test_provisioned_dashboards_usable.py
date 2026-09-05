"""Guard: every provisioned Grafana dashboard can bind Prometheus/Loki/Tempo.

Other dashboards besides HTTP API Latency were unusable because:
- Lightweight APM queried OTEL names (service_name, duration_seconds) while Tempo
  metrics-generator emits service + traces_spanmetrics_latency_*
- WeChat / data-archive used ${DS_PROMETHEUS} without a datasource variable
- Prometheus uid "Prometheus" (name) is not a valid datasource uid
"""

from __future__ import annotations

import json
from pathlib import Path

FILES = (
    Path(__file__).resolve().parents[1]
    / "grafana"
    / "provisioning"
    / "dashboards"
    / "files"
)


def _dashboards() -> list[tuple[str, dict]]:
    out = []
    for path in sorted(FILES.glob("*.json")):
        out.append((path.name, json.loads(path.read_text(encoding="utf-8"))))
    return out


def _walk_datasource_uids(obj: object) -> list[tuple[str | None, str]]:
    found: list[tuple[str | None, str]] = []
    if isinstance(obj, dict):
        ds = obj.get("datasource")
        if isinstance(ds, dict) and "uid" in ds:
            found.append((ds.get("type"), str(ds["uid"])))
        for v in obj.values():
            found.extend(_walk_datasource_uids(v))
    elif isinstance(obj, list):
        for v in obj:
            found.extend(_walk_datasource_uids(v))
    return found


def _variable_names(data: dict) -> set[str]:
    return {v.get("name", "") for v in data.get("templating", {}).get("list", [])}


def test_all_provisioned_dashboard_json_loads() -> None:
    names = [n for n, _ in _dashboards()]
    assert "http-api-latency.json" in names
    assert "lightweight-apm.json" in names
    assert "service-url-api-catalog.json" in names
    assert len(names) >= 8


def test_prometheus_uid_is_not_datasource_name() -> None:
    """Grafana interpolates uid, not name. uid \"Prometheus\" does not match PBFA97…."""
    for name, data in _dashboards():
        for dtype, uid in _walk_datasource_uids(data):
            if dtype == "prometheus":
                assert uid != "Prometheus", f"{name} uses Prometheus name as uid"


def test_ds_prometheus_template_exists_when_referenced() -> None:
    for name, data in _dashboards():
        uids = [uid for _, uid in _walk_datasource_uids(data)]
        if any(uid == "${DS_PROMETHEUS}" for uid in uids):
            assert "DS_PROMETHEUS" in _variable_names(data), (
                f"{name} references ${{DS_PROMETHEUS}} but has no datasource variable"
            )


def test_data_archive_uses_mysqld_exporter_v016_labels() -> None:
    data = json.loads(
        (FILES / "data-archive-monitoring.json").read_text(encoding="utf-8")
    )
    exprs = [
        t.get("expr", "")
        for p in data.get("panels") or []
        for t in (p.get("targets") or [])
    ]
    joined = "\n".join(exprs)
    assert "table_schema=" not in joined
    assert "table_name=" not in joined
    assert 'schema="task_bill"' in joined
    assert "table=~" in joined


def test_host_metrics_disk_queries_match_node_exporter_rootfs() -> None:
    data = json.loads((FILES / "host-metrics.json").read_text(encoding="utf-8"))
    exprs = [
        t.get("expr", "")
        for p in data.get("panels") or []
        for t in (p.get("targets") or [])
        if "node_filesystem" in t.get("expr", "")
    ]
    assert exprs
    for expr in exprs:
        assert 'mountpoint="/"' not in expr
        assert "/rootfs" in expr or "mountpoint=~" in expr


def test_prometheus_panels_have_explicit_datasource() -> None:
    """dashboardScene leaves panels without datasource as 'unavailable'."""
    skip_types = {"row", "text", "dashlist", "news", "welcome", "gettingstarted"}
    for name, data in _dashboards():
        if name.startswith("trace-") or name == "distributed-trace-view.json":
            continue
        for panel in data.get("panels") or []:
            if panel.get("type") in skip_types:
                continue
            if not (panel.get("targets") or []):
                continue
            ds = panel.get("datasource") or {}
            assert isinstance(ds, dict) and ds.get("uid"), (
                f"{name} panel {panel.get('title')!r} missing datasource uid"
            )
