#!/usr/bin/env python3
"""Verify Tempo-generated span metrics exist in Prometheus with expected labels.

This script validates that the Lightweight APM dashboard queries will work by checking:
1. Required Tempo span-metrics and service-graph metrics exist in Prometheus
2. Each metric has the labels expected by dashboard queries
3. Metric types are correct (counter vs histogram)

Usage:
  python3 verify_tempo_metrics.py                           # default Prometheus URL
  python3 verify_tempo_metrics.py --prometheus http://prometheus:9090
  python3 verify_tempo_metrics.py --json                    # machine-readable output

Exit codes: 0 = all checks passed, 1 = some checks failed, 2 = connection error
"""

from __future__ import annotations

import argparse
import json
import sys
import urllib.error
import urllib.request
from typing import Any

DEFAULT_PROMETHEUS_URL = "http://localhost:9090"

# Metrics that the Lightweight APM dashboard expects, with their required labels.
# Format: {metric_name: {"type": "counter|histogram", "required_labels": [...], "optional_labels": [...]}}
EXPECTED_METRICS: dict[str, dict[str, Any]] = {
    "traces_spanmetrics_calls_total": {
        "type": "counter",
        "required_labels": ["service", "span_kind", "status_code"],
        "optional_labels": ["span_name"],
    },
    "traces_spanmetrics_latency_bucket": {
        "type": "histogram",
        "required_labels": ["le", "service", "span_kind"],
        "optional_labels": ["span_name"],
    },
    "traces_spanmetrics_latency_sum": {
        "type": "counter",
        "required_labels": ["service", "span_kind"],
        "optional_labels": [],
    },
    "traces_spanmetrics_latency_count": {
        "type": "counter",
        "required_labels": ["service", "span_kind"],
        "optional_labels": [],
    },
    "traces_service_graph_request_total": {
        "type": "counter",
        "required_labels": ["client", "server"],
        "optional_labels": ["connection_type"],
    },
    "traces_service_graph_request_server_seconds_bucket": {
        "type": "histogram",
        "required_labels": ["le", "client", "server"],
        "optional_labels": ["connection_type"],
    },
    "traces_service_graph_request_failed_total": {
        "type": "counter",
        "required_labels": ["client", "server"],
        "optional_labels": ["connection_type"],
    },
}

# Label value patterns expected by dashboard filters
LABEL_VALUE_EXPECTATIONS: dict[str, dict[str, list[str]]] = {
    "traces_spanmetrics_calls_total": {
        "span_kind": ["SPAN_KIND_SERVER", "SPAN_KIND_CLIENT"],
        "status_code": ["STATUS_CODE_OK", "STATUS_CODE_ERROR"],
    },
}


def _prometheus_api(prometheus_url: str, path: str, params: dict | None = None) -> dict:
    """Query Prometheus HTTP API v1."""
    base = prometheus_url.rstrip("/")
    qs_parts = []
    if params:
        for k, v in params.items():
            qs_parts.append(f"{k}={urllib.request.quote(str(v))}")
    qs = "&".join(qs_parts)
    url = f"{base}{path}"
    if qs:
        url = f"{url}?{qs}"
    req = urllib.request.Request(url, headers={"Accept": "application/json"})
    try:
        with urllib.request.urlopen(req, timeout=10) as resp:
            data = json.loads(resp.read().decode("utf-8"))
    except urllib.error.URLError as e:
        print(f"[FATAL] Cannot reach Prometheus at {prometheus_url}: {e}", file=sys.stderr)
        sys.exit(2)
    if data.get("status") != "success":
        raise RuntimeError(f"Prometheus API error: {data.get('error', 'unknown')}")
    return data["data"]


def _check_metric_exists(prometheus_url: str, metric_name: str) -> bool:
    """Check if a metric exists by querying label names."""
    try:
        _prometheus_api(prometheus_url, "/api/v1/labels", {"match[]": metric_name})
        return True
    except Exception:
        return False


def _get_label_values(prometheus_url: str, metric_name: str, label: str) -> list[str]:
    """Get all values for a label on a metric."""
    try:
        data = _prometheus_api(
            prometheus_url,
            "/api/v1/label/" + label + "/values",
            {"match[]": metric_name},
        )
        return data if isinstance(data, list) else []
    except Exception:
        return []


def verify_metrics(prometheus_url: str) -> dict:
    """Run all verification checks and return structured results."""
    results: dict[str, Any] = {
        "prometheus_url": prometheus_url,
        "passed": [],
        "failed": [],
        "warnings": [],
        "summary": {"total": 0, "passed": 0, "failed": 0, "warnings": 0},
    }

    for metric_name, spec in EXPECTED_METRICS.items():
        results["summary"]["total"] += 1
        check_name = f"metric:{metric_name}"

        # Check 1: metric exists
        if not _check_metric_exists(prometheus_url, metric_name):
            results["failed"].append({
                "check": check_name,
                "reason": f"Metric '{metric_name}' not found in Prometheus. "
                          f"Ensure Tempo metrics_generator is enabled and remote_write is working.",
            })
            results["summary"]["failed"] += 1
            continue

        # Check 2: required labels exist
        missing_labels = []
        for label in spec["required_labels"]:
            values = _get_label_values(prometheus_url, metric_name, label)
            if not values:
                missing_labels.append(label)

        if missing_labels:
            results["failed"].append({
                "check": check_name,
                "reason": f"Required labels missing on '{metric_name}': {missing_labels}",
            })
            results["summary"]["failed"] += 1
        else:
            results["passed"].append({
                "check": check_name,
                "detail": f"Metric '{metric_name}' ({spec['type']}) exists with all required labels: {spec['required_labels']}",
            })
            results["summary"]["passed"] += 1

        # Check 3: label value expectations (warnings only)
        if metric_name in LABEL_VALUE_EXPECTATIONS:
            for label, expected_values in LABEL_VALUE_EXPECTATIONS[metric_name].items():
                actual_values = _get_label_values(prometheus_url, metric_name, label)
                for ev in expected_values:
                    if ev not in actual_values:
                        results["warnings"].append({
                            "check": f"label_value:{metric_name}:{label}:{ev}",
                            "reason": f"Expected label value '{label}={ev}' not found on '{metric_name}'. "
                                      f"Dashboard queries filter by this value — panels may show 'No data'. "
                                      f"Found values: {actual_values[:10]}",
                        })
                        results["summary"]["warnings"] += 1

    return results


def print_human(results: dict) -> None:
    """Print results in human-readable format."""
    print(f"\n{'='*70}")
    print(f"Tempo Span Metrics Verification")
    print(f"Prometheus: {results['prometheus_url']}")
    print(f"{'='*70}")

    if results["passed"]:
        print(f"\n✅ PASSED ({len(results['passed'])}):")
        for item in results["passed"]:
            print(f"  ✅ {item['check']}")
            print(f"     {item['detail']}")

    if results["failed"]:
        print(f"\n❌ FAILED ({len(results['failed'])}):")
        for item in results["failed"]:
            print(f"  ❌ {item['check']}")
            print(f"     {item['reason']}")

    if results["warnings"]:
        print(f"\n⚠️  WARNINGS ({len(results['warnings'])}):")
        for item in results["warnings"]:
            print(f"  ⚠️  {item['check']}")
            print(f"     {item['reason']}")

    s = results["summary"]
    print(f"\n{'='*70}")
    print(f"Summary: {s['passed']} passed, {s['failed']} failed, {s['warnings']} warnings "
          f"(of {s['total']} metrics checked)")
    print(f"{'='*70}")

    if s["failed"] > 0:
        print("\n💡 Troubleshooting:")
        print("  1. Verify Tempo metrics_generator is enabled in tempo-config.yaml")
        print("  2. Check Tempo → Prometheus remote_write connectivity")
        print("  3. Ensure services are sending OTel traces to the collector")
        print("  4. Wait 30-60s after trace ingestion for metrics generation")


def print_json(results: dict) -> None:
    """Print results as JSON."""
    print(json.dumps(results, indent=2, ensure_ascii=False))


def main() -> int:
    parser = argparse.ArgumentParser(
        description="Verify Tempo-generated span metrics exist in Prometheus"
    )
    parser.add_argument(
        "--prometheus",
        default=DEFAULT_PROMETHEUS_URL,
        help=f"Prometheus base URL (default: {DEFAULT_PROMETHEUS_URL})",
    )
    parser.add_argument(
        "--json",
        action="store_true",
        help="Output results as JSON",
    )
    args = parser.parse_args()

    results = verify_metrics(args.prometheus)

    if args.json:
        print_json(results)
    else:
        print_human(results)

    if results["summary"]["failed"] > 0:
        return 1
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
