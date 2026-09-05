"""Grafana dashboard JSON for the service URL/API catalog."""

from __future__ import annotations

from typing import Any

DASHBOARD_UID = "service-url-api-catalog"
DASHBOARD_TITLE = "Service URL / API Catalog"


def _prom_panel(pid: int, x: int, y: int, w: int, h: int, title: str, expr: str, ptype: str) -> dict[str, Any]:
    return {
        "datasource": {"type": "prometheus", "uid": "${DS_PROMETHEUS}"},
        "fieldConfig": {"defaults": {}, "overrides": []},
        "gridPos": {"h": h, "w": w, "x": x, "y": y},
        "id": pid,
        "targets": [
            {
                "datasource": {"type": "prometheus", "uid": "${DS_PROMETHEUS}"},
                "editorMode": "code",
                "expr": expr,
                "legendFormat": "{{service}} {{path}}",
                "refId": "A",
            }
        ],
        "title": title,
        "type": ptype,
    }


def render_grafana_dashboard(catalog: dict[str, Any]) -> dict[str, Any]:
    s = catalog.get("summary") or {}
    by = catalog.get("by_service") or {}
    summary_md = (
        "### Intended catalog (config SSOT) + runtime overlay\n\n"
        "Grafana here is the **browse + runtime** layer. Intended URLs come from "
        "`routes.yaml` + `api_route_ownership.yaml` + taskFE router — regenerate with "
        "`python3 db/scripts/ci/build_service_url_api_catalog.py`.\n\n"
        f"- API routes: **{s.get('api_routes', 0)}** (core **{s.get('core_api', 0)}**, "
        f"orphaned **{s.get('orphaned_api', 0)}**)\n"
        f"- SPA pages: **{s.get('fe_pages', 0)}** (core **{s.get('core_fe', 0)}**)\n"
        f"- Upstreams with routes: **{s.get('services', 0)}**\n\n"
        "Left: Tempo **service map** = observed call graph. Right/below: Prometheus "
        "request rate = empirical hot paths. Declared core ≠ high QPS.\n"
    )
    table_md_rows = [
        "| service | core | supporting | ops | internal | orphaned |",
        "|---|---:|---:|---:|---:|---:|",
    ]
    for svc, counts in list(by.items())[:40]:
        table_md_rows.append(
            f"| {svc} | {counts.get('core', 0)} | {counts.get('supporting', 0)} | "
            f"{counts.get('ops', 0)} | {counts.get('internal', 0)} | {counts.get('orphaned', 0)} |"
        )
    core_lines = ["| uri | upstream | stream |", "|---|---|---|"]
    for r in catalog.get("api_routes") or []:
        if r.get("role") != "core":
            continue
        core_lines.append(f"| {r.get('uri')} | {r.get('upstream')} | {r.get('value_stream') or ''} |")
    panels: list[dict[str, Any]] = [
        {
            "gridPos": {"h": 8, "w": 24, "x": 0, "y": 0},
            "id": 1,
            "options": {"content": summary_md, "mode": "markdown"},
            "title": "Catalog decision",
            "type": "text",
        },
        {
            "datasource": {"type": "tempo", "uid": "tempo"},
            "gridPos": {"h": 14, "w": 12, "x": 0, "y": 8},
            "id": 2,
            "options": {"nodes": {}},
            "targets": [{"datasource": {"type": "tempo", "uid": "tempo"}, "queryType": "serviceMap", "refId": "A"}],
            "title": "Runtime service map (Tempo)",
            "type": "nodeGraph",
        },
        _prom_panel(
            3,
            12,
            8,
            12,
            7,
            "Empirical hot paths (HTTP rate by path)",
            'topk(15, sum by (service, path) (rate({__name__=~".*_http_requests_total"}[$__rate_interval])))',
            "table",
        ),
        _prom_panel(
            4,
            12,
            15,
            12,
            7,
            "Request rate by service",
            'sum by (service) (rate({__name__=~".*_http_requests_total"}[$__rate_interval]))',
            "timeseries",
        ),
        {
            "gridPos": {"h": 12, "w": 12, "x": 0, "y": 22},
            "id": 5,
            "options": {"content": "### By upstream\n\n" + "\n".join(table_md_rows), "mode": "markdown"},
            "title": "Intended routes by service",
            "type": "text",
        },
        {
            "gridPos": {"h": 12, "w": 12, "x": 12, "y": 22},
            "id": 6,
            "options": {"content": "### Declared core APIs\n\n" + "\n".join(core_lines[:80]), "mode": "markdown"},
            "title": "Declared core API prefixes",
            "type": "text",
        },
    ]
    return {
        "annotations": {"list": []},
        "editable": True,
        "fiscalYearStartMonth": 0,
        "graphTooltip": 1,
        "id": None,
        "links": [
            {"icon": "dashboard", "title": "HTTP API Latency", "type": "link", "url": "/d/http-api-latency/http-api-latency"},
            {"icon": "dashboard", "title": "Lightweight APM", "type": "link", "url": "/d/lightweight-apm/lightweight-apm"},
            {
                "icon": "dashboard",
                "title": "Distributed Trace View",
                "type": "link",
                "url": "/d/distributed-trace-view/distributed-trace-view",
            },
        ],
        "panels": panels,
        "refresh": "30s",
        "schemaVersion": 39,
        "tags": ["aimonitor", "catalog", "gateway", "core-path"],
        "templating": {
            "list": [
                {
                    "current": {"text": "Prometheus", "value": "Prometheus"},
                    "hide": 2,
                    "name": "DS_PROMETHEUS",
                    "query": "prometheus",
                    "refresh": 1,
                    "type": "datasource",
                }
            ]
        },
        "time": {"from": "now-6h", "to": "now"},
        "timezone": "browser",
        "title": DASHBOARD_TITLE,
        "uid": DASHBOARD_UID,
        "version": 1,
    }
