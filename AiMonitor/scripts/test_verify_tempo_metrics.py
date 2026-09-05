"""Test verify_tempo_metrics.py — Prometheus API mock-based tests."""

from __future__ import annotations

import json
import sys
from pathlib import Path
from unittest.mock import MagicMock, patch

# Ensure the scripts directory is importable
sys.path.insert(0, str(Path(__file__).resolve().parents[1] / "scripts"))

import verify_tempo_metrics as vtm


class TestExpectedMetricsDefinition:
    """Validate EXPECTED_METRICS structure is correct."""

    def test_all_required_metrics_defined(self) -> None:
        required = [
            "traces_spanmetrics_calls_total",
            "traces_spanmetrics_latency_bucket",
            "traces_service_graph_request_total",
            "traces_service_graph_request_server_seconds_bucket",
        ]
        for name in required:
            assert name in vtm.EXPECTED_METRICS, f"Missing required metric: {name}"

    def test_span_metrics_have_service_label(self) -> None:
        for name, spec in vtm.EXPECTED_METRICS.items():
            if "spanmetrics" in name and "service_graph" not in name:
                assert "service" in spec["required_labels"], (
                    f"{name} must require Tempo generator service label"
                )
                assert "service_name" not in spec["required_labels"]

    def test_service_graph_metrics_have_client_server_labels(self) -> None:
        for name, spec in vtm.EXPECTED_METRICS.items():
            if "service_graph" in name:
                assert "client" in spec["required_labels"], (
                    f"{name} must require client label"
                )
                assert "server" in spec["required_labels"], (
                    f"{name} must require server label"
                )

    def test_histogram_metrics_have_le_label(self) -> None:
        for name, spec in vtm.EXPECTED_METRICS.items():
            if spec["type"] == "histogram":
                assert "le" in spec["required_labels"], (
                    f"Histogram {name} must require le label"
                )

    def test_label_value_expectations_cover_key_metrics(self) -> None:
        assert "traces_spanmetrics_calls_total" in vtm.LABEL_VALUE_EXPECTATIONS
        expectations = vtm.LABEL_VALUE_EXPECTATIONS["traces_spanmetrics_calls_total"]
        assert "span_kind" in expectations
        assert "status_code" in expectations
        assert "SPAN_KIND_SERVER" in expectations["span_kind"]
        assert "STATUS_CODE_ERROR" in expectations["status_code"]


class TestVerifyMetricsLogic:
    """Test verification logic with mocked Prometheus API."""

    def test_all_metrics_exist_and_pass(self) -> None:
        """When all metrics exist with all labels, everything passes."""
        with (
            patch.object(vtm, "_check_metric_exists", return_value=True),
            patch.object(vtm, "_get_label_values", return_value=["service-a", "service-b"]),
        ):
            results = vtm.verify_metrics("http://prom:9090")

        assert results["summary"]["passed"] > 0
        assert results["summary"]["failed"] == 0

    def test_missing_metric_reports_failure(self) -> None:
        """When a metric is missing, it should be reported as failed."""
        def mock_exists(prom_url, metric_name):
            # First metric fails, others succeed
            if metric_name == "traces_spanmetrics_calls_total":
                return False
            return True

        with (
            patch.object(vtm, "_check_metric_exists", side_effect=mock_exists),
            patch.object(vtm, "_get_label_values", return_value=["svc"]),
        ):
            results = vtm.verify_metrics("http://prom:9090")

        assert results["summary"]["failed"] >= 1
        assert any("traces_spanmetrics_calls_total" in str(f) for f in results["failed"])

    def test_missing_label_reports_failure(self) -> None:
        """When a required label has no values, report failure."""
        def mock_labels(prom_url, metric_name, label):
            if label == "service":
                return []  # empty = missing
            return ["ok"]

        with (
            patch.object(vtm, "_check_metric_exists", return_value=True),
            patch.object(vtm, "_get_label_values", side_effect=mock_labels),
        ):
            results = vtm.verify_metrics("http://prom:9090")

        assert results["summary"]["failed"] >= 1
        assert any("service" in str(f) for f in results["failed"])

    def test_missing_label_value_generates_warning(self) -> None:
        """When expected label value isn't found, generate a warning (not failure)."""
        def mock_labels(prom_url, metric_name, label):
            if label == "span_kind":
                return ["OTHER_KIND"]  # missing SPAN_KIND_SERVER
            return ["ok"]

        with (
            patch.object(vtm, "_check_metric_exists", return_value=True),
            patch.object(vtm, "_get_label_values", side_effect=mock_labels),
        ):
            results = vtm.verify_metrics("http://prom:9090")

        assert results["summary"]["warnings"] >= 1
        assert any("SPAN_KIND_SERVER" in str(w) for w in results["warnings"])
        # Metrics still pass if labels exist, even if specific values are missing
        assert results["summary"]["passed"] > 0

    def test_connection_error_exit_code(self) -> None:
        """Connection errors should produce exit code 2."""
        with patch.object(vtm, "_prometheus_api", side_effect=SystemExit(2)):
            try:
                vtm.verify_metrics("http://prom:9090")
            except SystemExit as e:
                assert e.code == 2
                return
        assert False, "Should have raised SystemExit(2)"

    def test_result_structure(self) -> None:
        """Verify result dict has all expected fields."""
        with (
            patch.object(vtm, "_check_metric_exists", return_value=True),
            patch.object(vtm, "_get_label_values", return_value=["val"]),
        ):
            results = vtm.verify_metrics("http://prom:9090")

        assert "prometheus_url" in results
        assert "passed" in results
        assert "failed" in results
        assert "warnings" in results
        assert "summary" in results
        assert "total" in results["summary"]
        assert "passed" in results["summary"]
        assert "failed" in results["summary"]
        assert "warnings" in results["summary"]
        for item in results["passed"]:
            assert "check" in item
            assert "detail" in item
        for item in results["failed"]:
            assert "check" in item
            assert "reason" in item


class TestMainFunction:
    """Test main() return codes."""

    def test_main_returns_0_when_no_failures(self) -> None:
        with (
            patch.object(vtm, "verify_metrics", return_value={
                "summary": {"passed": 7, "failed": 0, "warnings": 0, "total": 7},
                "passed": [], "failed": [], "warnings": [],
                "prometheus_url": "http://test",
            }),
            patch("sys.argv", ["verify_tempo_metrics.py"]),
        ):
            assert vtm.main() == 0

    def test_main_returns_1_when_failures_exist(self) -> None:
        with (
            patch.object(vtm, "verify_metrics", return_value={
                "summary": {"passed": 5, "failed": 2, "warnings": 0, "total": 7},
                "passed": [], "failed": [], "warnings": [],
                "prometheus_url": "http://test",
            }),
            patch("sys.argv", ["verify_tempo_metrics.py"]),
        ):
            assert vtm.main() == 1
