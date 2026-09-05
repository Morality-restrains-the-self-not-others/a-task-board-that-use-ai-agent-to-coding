#!/usr/bin/env python3
"""Periodically query Loki for per-service log stats and push collection_status entries.

These collection_status entries power the Trace Log Explore dashboard panels:
  - "Active Log Sources"   — count of unique services with collection_status in last 1m
  - "Log Collection Status (per service)" — table of per-service stats

Usage:
  python3 log_collection_status_reporter.py                    # one-shot
  python3 log_collection_status_reporter.py --interval 60      # loop every 60s
  python3 log_collection_status_reporter.py --loki http://loki:3100 --dry-run

The script queries Loki for:
  1. All service labels via /loki/api/v1/label/service/values
  2. Per-service log counts via count_over_time (last 1m)
  3. Per-service last log timestamp

Then pushes a single collection_status log stream back to Loki with one entry per service
(plus a summary entry), so the dashboard stat + table panels render.
"""

from __future__ import annotations

import argparse
import json
import os
import sys
import time
import urllib.error
import urllib.parse
import urllib.request
from datetime import datetime, timezone
from typing import Any


LOKI_PUSH_URL_DEFAULT = "http://localhost:3100/loki/api/v1/push"
LOKI_QUERY_URL_DEFAULT = "http://localhost:3100/loki/api/v1"
JOB_NAME = "log-collection-status"
SERVICE_NAME = "log-collection-status"
SLEEP_BETWEEN_REQUESTS = 0.05  # 50ms to avoid hammering Loki

# Prometheus textfile collector output path (for node_exporter)
PROM_TEXTFILE_DIR = os.environ.get("PROM_TEXTFILE_DIR", "/var/lib/prometheus/node-exporter")


class LokiClient:
    """Minimal Loki HTTP client — no third-party deps."""

    def __init__(self, base_url: str):
        base = base_url.rstrip("/")
        # OPT-20260831-020：只传主机（如 compose --loki http://aimonitor-loki:3100）时
        # 自动补 Loki API 根路径，避免查询落到 .../label/... 404。
        if not urllib.parse.urlsplit(base).path:
            base += "/loki/api/v1"
        self.base_url = base.rstrip("/")

    def _get(self, path: str, params: dict | None = None) -> dict[str, Any]:
        if params:
            qs = "&".join(f"{k}={urllib.request.quote(str(v))}" for k, v in params.items())
            url = f"{self.base_url}{path}?{qs}"
        else:
            url = f"{self.base_url}{path}"
        req = urllib.request.Request(url, headers={"Accept": "application/json"})
        try:
            with urllib.request.urlopen(req, timeout=15) as resp:
                data = json.loads(resp.read().decode("utf-8"))
        except urllib.error.URLError as exc:
            print(f"[ERROR] Loki query failed: {url} — {exc}", file=sys.stderr)
            raise
        return data  # type: ignore[no-any-return]

    def _post(self, path: str, body: bytes) -> int:
        url = f"{self.base_url}{path}"
        req = urllib.request.Request(
            url,
            data=body,
            headers={"Content-Type": "application/json"},
            method="POST",
        )
        try:
            with urllib.request.urlopen(req, timeout=15) as resp:
                return resp.status
        except urllib.error.URLError as exc:
            print(f"[ERROR] Loki push failed: {url} — {exc}", file=sys.stderr)
            raise

    def list_services(self) -> list[str]:
        """Return all distinct service label values known to Loki."""
        data = self._get("/label/service/values")
        # Loki returns {"status":"success","data":["svc1","svc2",...]}
        if data.get("status") != "success":
            print(f"[WARN] Unexpected Loki response: {data}", file=sys.stderr)
            return []
        return sorted(data.get("data", []))

    def count_lines_by_service(self, window: str = "1m") -> dict[str, int]:
        """Return per-service log counts in a single query using sum by (service)."""
        query = f'sum by (service) (count_over_time({{job=~".+"}}[{window}]))'
        now_ns = int(time.time() * 1e9)
        start_ns = now_ns - 120_000_000_000  # 2 min ago
        data = self._get(
            "/query_range",
            {
                "query": query,
                "start": str(start_ns),
                "end": str(now_ns),
                "step": "60",
            },
        )
        results = data.get("data", {}).get("result", [])
        out: dict[str, int] = {}
        for r in results:
            svc = (r.get("metric", {}) or {}).get("service", "")
            if not svc:
                continue
            values = r.get("values", [])
            if values:
                try:
                    out[svc] = int(float(values[-1][1]))
                except (ValueError, IndexError, TypeError):
                    out[svc] = 0
            else:
                out[svc] = 0
        return out

    def count_lines(self, service: str, window: str = "1m") -> int:
        """Count log lines for a single service in the given time window (fallback)."""
        query = f'count_over_time({{service="{service}"}}[{window}])'
        now_ns = int(time.time() * 1e9)
        start_ns = now_ns - 120_000_000_000  # 2 min ago
        data = self._get(
            "/query_range",
            {
                "query": query,
                "start": str(start_ns),
                "end": str(now_ns),
                "step": "60",
            },
        )
        results = data.get("data", {}).get("result", [])
        if not results:
            return 0
        # Take the last value from the first (only) result
        values = results[0].get("values", [])
        if not values:
            return 0
        try:
            return int(float(values[-1][1]))
        except (ValueError, IndexError, TypeError):
            return 0

    def last_log_ns(self, service: str) -> int:
        """Return nanosecond timestamp of the most recent log for a service."""
        query = f'{{service="{service}"}}'
        # Loki 3.x rejects a bare log stream selector on the instant /query endpoint
        # ("log queries are not supported as an instant query type"), so use /query_range.
        # 24h lookback bounds index cost while covering dead-threshold horizons.
        now_ns = int(time.time() * 1e9)
        data = self._get(
            "/query_range",
            {
                "query": query,
                "start": str(now_ns - 24 * 3600 * 10**9),
                "end": str(now_ns),
                "limit": "1",
                "direction": "backward",
            },
        )
        results = data.get("data", {}).get("result", [])
        if not results:
            return 0
        values = results[0].get("values", [])
        if not values:
            return 0
        try:
            return int(values[0][0])
        except (ValueError, IndexError, TypeError):
            return 0

    def push_collection_status(self, entries: list[dict[str, Any]]) -> bool:
        """Push collection_status log entries to Loki."""
        now_ns = str(int(time.time() * 1e9))
        values: list[list[str]] = []
        for entry in entries:
            values.append([now_ns, json.dumps(entry, ensure_ascii=False)])

        payload = {
            "streams": [
                {
                    "stream": {
                        "job": JOB_NAME,
                        "service": SERVICE_NAME,
                        "msg": "collection_status",
                    },
                    "values": values,
                }
            ]
        }
        body = json.dumps(payload, ensure_ascii=False).encode("utf-8")
        status = self._post("/push", body)
        return 200 <= status < 300


def parse_args() -> argparse.Namespace:
    parser = argparse.ArgumentParser(
        description="Push per-service log collection status to Loki for Grafana dashboard panels."
    )
    parser.add_argument(
        "--loki",
        default=os.environ.get("LOKI_URL", LOKI_QUERY_URL_DEFAULT),
        help=f"Loki base URL (default: {LOKI_QUERY_URL_DEFAULT})",
    )
    parser.add_argument(
        "--interval",
        type=int,
        default=0,
        help="Run periodically every N seconds (0 = one-shot, default)",
    )
    parser.add_argument(
        "--dry-run",
        action="store_true",
        help="Print what would be pushed without actually pushing",
    )
    parser.add_argument(
        "--timeout",
        type=int,
        default=30,
        help="Per-request HTTP timeout in seconds (default: 30)",
    )
    parser.add_argument(
        "--stale-threshold-minutes",
        type=int,
        default=5,
        help="Services with last log older than this are marked 'stale' (default: 5)",
    )
    parser.add_argument(
        "--dead-threshold-minutes",
        type=int,
        default=30,
        help="Services with last log older than this are marked 'dead' (default: 30)",
    )
    return parser.parse_args()


def determine_status(
    last_log_ns: int,
    stale_threshold_min: int,
    dead_threshold_min: int,
) -> str:
    """Classify service status based on log recency."""
    if last_log_ns == 0:
        return "unknown"
    now_ns = time.time() * 1e9
    age_seconds = (now_ns - last_log_ns) / 1e9
    if age_seconds > dead_threshold_min * 60:
        return "dead"
    if age_seconds > stale_threshold_min * 60:
        return "stale"
    return "healthy"


def write_prom_textfile(entries: list[dict[str, Any]]) -> bool:
    """Write Prometheus textfile for node_exporter textfile collector.

    Exposes:
      log_collection_services_total{status="healthy|stale|dead|unknown"} <count>
      log_collection_lines_total <sum>
    """
    if not PROM_TEXTFILE_DIR or not os.path.isdir(PROM_TEXTFILE_DIR):
        return False
    status_counts: dict[str, int] = {"healthy": 0, "stale": 0, "dead": 0, "unknown": 0}
    total_lines = 0
    for e in entries:
        s = str(e.get("service_status", "unknown")).lower()
        if s in status_counts:
            status_counts[s] += 1
        else:
            status_counts["unknown"] += 1
        total_lines += int(e.get("total_lines", 0))
    lines = [
        "# HELP log_collection_services_total Number of services by log collection status.",
        "# TYPE log_collection_services_total gauge",
    ]
    for status, count in status_counts.items():
        lines.append(f'log_collection_services_total{{status="{status}"}} {count}')
    lines.append("# HELP log_collection_lines_total Total log lines/min across all services.")
    lines.append("# TYPE log_collection_lines_total gauge")
    lines.append(f"log_collection_lines_total {total_lines}")
    lines.append("")  # trailing newline
    path = os.path.join(PROM_TEXTFILE_DIR, "log_collection_status.prom")
    tmp = path + ".tmp"
    try:
        with open(tmp, "w") as f:
            f.write("\n".join(lines))
        os.replace(tmp, path)
        return True
    except OSError as exc:
        print(f"[WARN] Failed to write prom textfile: {exc}", file=sys.stderr)
        return False


def format_last_log_at(last_log_ns: int) -> str:
    """Format nanosecond timestamp as ISO 8601 UTC string."""
    if last_log_ns == 0:
        return "N/A"
    dt = datetime.fromtimestamp(last_log_ns / 1e9, tz=timezone.utc)
    return dt.strftime("%Y-%m-%dT%H:%M:%SZ")


def run_reporter(
    loki_url: str,
    dry_run: bool,
    stale_threshold_min: int,
    dead_threshold_min: int,
) -> dict[str, Any]:
    """Execute one reporting cycle."""
    client = LokiClient(loki_url)

    services = client.list_services()
    if not services:
        print("[INFO] No services found in Loki. Is Promtail running and sending logs?")
        if not dry_run:
            # Push a "no services" status entry so the dashboard isn't empty
            client.push_collection_status(
                [
                    {
                        "service": "(none)",
                        "total_lines": 0,
                        "service_status": "unknown",
                        "last_log_at": "N/A",
                    }
                ]
            )
        return {"services": 0, "lines": 0}

    entries: list[dict[str, Any]] = []
    total_lines = 0

    # Batch-fetch per-service log counts in a single Loki query (OPT-003)
    count_by_svc = client.count_lines_by_service()

    for svc in services:
        count = count_by_svc.get(svc, 0)
        total_lines += count
        last_ns = client.last_log_ns(svc)
        status = determine_status(last_ns, stale_threshold_min, dead_threshold_min)
        last_at = format_last_log_at(last_ns)

        entries.append(
            {
                "service": svc,
                "total_lines": count,
                "service_status": status,
                "last_log_at": last_at,
            }
        )
        # Small sleep to avoid overwhelming Loki
        if len(entries) < len(services):
            time.sleep(SLEEP_BETWEEN_REQUESTS)

    # Sort: healthy first, then by total_lines desc
    status_order = {"healthy": 0, "stale": 1, "dead": 2, "unknown": 3}
    entries.sort(key=lambda e: (status_order.get(e["service_status"], 4), -e["total_lines"]))

    if dry_run:
        print(f"[DRY-RUN] Would push {len(entries)} service status entries:")
        for e in entries:
            print(
                f"  {e['service']:40s} lines={e['total_lines']:>6d}  "
                f"status={e['service_status']:8s}  last={e['last_log_at']}"
            )
        return {"services": len(services), "lines": total_lines}

    ok = client.push_collection_status(entries)
    # Write Prometheus textfile for node_exporter — enables LogSourceDead alerting (OPT-004)
    write_prom_textfile(entries)
    if ok:
        print(
            f"[OK  ] Pushed {len(entries)} service status entries "
            f"({total_lines} total lines/min across {len(services)} services)"
        )
    else:
        print(f"[FAIL] Loki push returned non-2xx status", file=sys.stderr)
    return {"services": len(services), "lines": total_lines}


def main() -> int:
    args = parse_args()
    loki_url = args.loki.rstrip("/")

    if args.interval > 0:
        print(f"[LOOP] Reporting every {args.interval}s to {loki_url}")
        # Initial delay to let the stack settle
        time.sleep(5)
        while True:
            try:
                run_reporter(
                    loki_url,
                    dry_run=args.dry_run,
                    stale_threshold_min=args.stale_threshold_minutes,
                    dead_threshold_min=args.dead_threshold_minutes,
                )
            except Exception as exc:
                print(f"[ERROR] Reporter cycle failed: {exc}", file=sys.stderr)
            time.sleep(args.interval)
    else:
        try:
            run_reporter(
                loki_url,
                dry_run=args.dry_run,
                stale_threshold_min=args.stale_threshold_minutes,
                dead_threshold_min=args.dead_threshold_minutes,
            )
        except Exception as exc:
            print(f"[ERROR] {exc}", file=sys.stderr)
            return 1
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
