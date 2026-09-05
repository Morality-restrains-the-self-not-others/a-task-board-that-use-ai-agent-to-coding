"""Tests for log_collection_status_reporter.py."""

from __future__ import annotations

import json
import time
import unittest
from unittest.mock import MagicMock, call, patch

from log_collection_status_reporter import (
    JOB_NAME,
    SERVICE_NAME,
    LokiClient,
    determine_status,
    format_last_log_at,
    run_reporter,
)


class TestDetermineStatus(unittest.TestCase):
    """Unit tests for service status classification."""

    def test_healthy(self) -> None:
        now_ns = int(time.time() * 1e9)
        last_ns = now_ns - int(2 * 60 * 1e9)  # 2 min ago
        assert determine_status(last_ns, stale_threshold_min=5, dead_threshold_min=30) == "healthy"

    def test_stale(self) -> None:
        now_ns = int(time.time() * 1e9)
        last_ns = now_ns - int(10 * 60 * 1e9)  # 10 min ago
        assert determine_status(last_ns, stale_threshold_min=5, dead_threshold_min=30) == "stale"

    def test_dead(self) -> None:
        now_ns = int(time.time() * 1e9)
        last_ns = now_ns - int(60 * 60 * 1e9)  # 1 hour ago
        assert determine_status(last_ns, stale_threshold_min=5, dead_threshold_min=30) == "dead"

    def test_unknown(self) -> None:
        assert determine_status(0, stale_threshold_min=5, dead_threshold_min=30) == "unknown"

    def test_boundary_stale(self) -> None:
        now_ns = int(time.time() * 1e9)
        # Exactly at the stale threshold boundary (> stale, <= dead)
        last_ns = now_ns - int(5 * 60 * 1e9 + 1)  # just over 5 min
        assert determine_status(last_ns, stale_threshold_min=5, dead_threshold_min=30) == "stale"

    def test_boundary_dead(self) -> None:
        now_ns = int(time.time() * 1e9)
        last_ns = now_ns - int(30 * 60 * 1e9 + 1)  # just over 30 min
        assert determine_status(last_ns, stale_threshold_min=5, dead_threshold_min=30) == "dead"


class TestFormatLastLogAt(unittest.TestCase):
    """Unit tests for timestamp formatting."""

    def test_zero(self) -> None:
        assert format_last_log_at(0) == "N/A"

    def test_known_timestamp(self) -> None:
        # 2026-07-24T10:00:00Z in nanoseconds
        import datetime as dt_mod
        expected_dt = dt_mod.datetime(2026, 7, 24, 10, 0, 0, tzinfo=dt_mod.timezone.utc)
        ts_ns = int(expected_dt.timestamp() * 1e9)
        result = format_last_log_at(ts_ns)
        assert result == "2026-07-24T10:00:00Z"


class TestLokiClient(unittest.TestCase):
    """Unit tests for LokiClient with mocked HTTP."""

    def setUp(self) -> None:
        self.client = LokiClient("http://loki:3100")

    @patch("log_collection_status_reporter.urllib.request.urlopen")
    def test_list_services_success(self, mock_urlopen: MagicMock) -> None:
        mock_resp = MagicMock()
        mock_resp.read.return_value = json.dumps(
            {"status": "success", "data": ["svc-a", "svc-b", "svc-c"]}
        ).encode()
        mock_resp.__enter__.return_value = mock_resp
        mock_urlopen.return_value = mock_resp

        services = self.client.list_services()
        assert services == ["svc-a", "svc-b", "svc-c"]

    @patch("log_collection_status_reporter.urllib.request.urlopen")
    def test_list_services_empty(self, mock_urlopen: MagicMock) -> None:
        mock_resp = MagicMock()
        mock_resp.read.return_value = json.dumps(
            {"status": "success", "data": []}
        ).encode()
        mock_resp.__enter__.return_value = mock_resp
        mock_urlopen.return_value = mock_resp

        services = self.client.list_services()
        assert services == []

    @patch("log_collection_status_reporter.urllib.request.urlopen")
    def test_base_url_without_api_prefix_appends_loki_api_v1(self, mock_urlopen: MagicMock) -> None:
        """只传主机 base（OPT-20260831-020）：查询路径必须落到 /loki/api/v1。"""
        client = LokiClient("http://loki:3100")
        mock_resp = MagicMock()
        mock_resp.read.return_value = json.dumps({"status": "success", "data": []}).encode()
        mock_resp.__enter__.return_value = mock_resp
        mock_urlopen.return_value = mock_resp

        client.list_services()

        req = mock_urlopen.call_args[0][0]
        assert "/loki/api/v1/label/service/values" in req.full_url

    @patch("log_collection_status_reporter.urllib.request.urlopen")
    def test_base_url_with_api_prefix_not_duplicated(self, mock_urlopen: MagicMock) -> None:
        """已带 /loki/api/v1 的 base 不得二次拼接。"""
        client = LokiClient("http://loki:3100/loki/api/v1")
        mock_resp = MagicMock()
        mock_resp.read.return_value = json.dumps({"status": "success", "data": []}).encode()
        mock_resp.__enter__.return_value = mock_resp
        mock_urlopen.return_value = mock_resp

        client.list_services()

        req = mock_urlopen.call_args[0][0]
        assert req.full_url == "http://loki:3100/loki/api/v1/label/service/values"

    @patch("log_collection_status_reporter.urllib.request.urlopen")
    def test_count_lines_with_data(self, mock_urlopen: MagicMock) -> None:
        mock_resp = MagicMock()
        mock_resp.read.return_value = json.dumps(
            {
                "status": "success",
                "data": {
                    "result": [
                        {"values": [["1711756800000000000", "42"]]}
                    ]
                },
            }
        ).encode()
        mock_resp.__enter__.return_value = mock_resp
        mock_urlopen.return_value = mock_resp

        count = self.client.count_lines("test-svc")
        assert count == 42

    @patch("log_collection_status_reporter.urllib.request.urlopen")
    def test_count_lines_empty(self, mock_urlopen: MagicMock) -> None:
        mock_resp = MagicMock()
        mock_resp.read.return_value = json.dumps(
            {"status": "success", "data": {"result": []}}
        ).encode()
        mock_resp.__enter__.return_value = mock_resp
        mock_urlopen.return_value = mock_resp

        count = self.client.count_lines("test-svc")
        assert count == 0

    @patch("log_collection_status_reporter.urllib.request.urlopen")
    def test_last_log_ns_with_data(self, mock_urlopen: MagicMock) -> None:
        mock_resp = MagicMock()
        mock_resp.read.return_value = json.dumps(
            {
                "status": "success",
                "data": {
                    "result": [
                        {"values": [["1785117600000000000", '{"msg":"test"}']]}
                    ]
                },
            }
        ).encode()
        mock_resp.__enter__.return_value = mock_resp
        mock_urlopen.return_value = mock_resp

        last_ns = self.client.last_log_ns("test-svc")
        assert last_ns == 1785117600000000000

    @patch("log_collection_status_reporter.urllib.request.urlopen")
    def test_last_log_ns_uses_query_range_not_instant_query(self, mock_urlopen: MagicMock) -> None:
        """Loki 3.x rejects bare stream selectors on instant /query — must use /query_range."""
        mock_resp = MagicMock()
        mock_resp.read.return_value = json.dumps(
            {
                "status": "success",
                "data": {
                    "result": [
                        {"values": [["1785117600000000000", '{"msg":"test"}']]}
                    ]
                },
            }
        ).encode()
        mock_resp.__enter__.return_value = mock_resp
        mock_urlopen.return_value = mock_resp

        last_ns = self.client.last_log_ns("test-svc")
        assert last_ns == 1785117600000000000

        req = mock_urlopen.call_args[0][0]
        assert "/loki/api/v1/query_range" in req.full_url
        assert "direction=backward" in req.full_url
        assert "start=" in req.full_url
        assert "end=" in req.full_url

    @patch("log_collection_status_reporter.urllib.request.urlopen")
    def test_last_log_ns_empty(self, mock_urlopen: MagicMock) -> None:
        mock_resp = MagicMock()
        mock_resp.read.return_value = json.dumps(
            {"status": "success", "data": {"result": []}}
        ).encode()
        mock_resp.__enter__.return_value = mock_resp
        mock_urlopen.return_value = mock_resp

        last_ns = self.client.last_log_ns("test-svc")
        assert last_ns == 0

    @patch("log_collection_status_reporter.urllib.request.urlopen")
    def test_push_collection_status(self, mock_urlopen: MagicMock) -> None:
        mock_resp = MagicMock()
        mock_resp.status = 204
        mock_resp.__enter__.return_value = mock_resp
        mock_urlopen.return_value = mock_resp

        entries = [
            {
                "service": "svc-a",
                "total_lines": 10,
                "service_status": "healthy",
                "last_log_at": "2026-07-24T10:00:00Z",
            }
        ]
        ok = self.client.push_collection_status(entries)
        assert ok is True

        # Verify the pushed payload structure
        call_args = mock_urlopen.call_args
        req = call_args[0][0]  # The Request object
        body = json.loads(req.data)
        streams = body["streams"]
        assert len(streams) == 1
        stream = streams[0]["stream"]
        assert stream["job"] == JOB_NAME
        assert stream["service"] == SERVICE_NAME
        assert stream["msg"] == "collection_status"
        values = streams[0]["values"]
        assert len(values) == 1
        log_entry = json.loads(values[0][1])
        assert log_entry["service"] == "svc-a"
        assert log_entry["total_lines"] == 10


class TestRunReporterIntegration(unittest.TestCase):
    """Integration-style tests for run_reporter with mocked LokiClient."""

    @patch("log_collection_status_reporter.LokiClient")
    def test_with_services(self, mock_client_cls: MagicMock) -> None:
        mock_client = MagicMock()
        mock_client.list_services.return_value = ["svc-a", "svc-b"]
        mock_client.count_lines_by_service.return_value = {"svc-a": 100, "svc-b": 50}
        mock_client.last_log_ns.side_effect = [
            int(time.time() * 1e9) - int(2 * 60 * 1e9),  # healthy
            int(time.time() * 1e9) - int(10 * 60 * 1e9),  # stale
        ]
        mock_client.push_collection_status.return_value = True
        mock_client_cls.return_value = mock_client

        result = run_reporter(
            loki_url="http://loki:3100",
            dry_run=False,
            stale_threshold_min=5,
            dead_threshold_min=30,
        )

        assert result["services"] == 2
        assert result["lines"] == 150
        mock_client.push_collection_status.assert_called_once()

        # Verify entries are sorted: healthy before stale
        entries = mock_client.push_collection_status.call_args[0][0]
        assert entries[0]["service"] == "svc-a"
        assert entries[0]["service_status"] == "healthy"
        assert entries[1]["service"] == "svc-b"
        assert entries[1]["service_status"] == "stale"

    @patch("log_collection_status_reporter.LokiClient")
    def test_no_services(self, mock_client_cls: MagicMock) -> None:
        mock_client = MagicMock()
        mock_client.list_services.return_value = []
        mock_client.push_collection_status.return_value = True
        mock_client_cls.return_value = mock_client

        result = run_reporter(
            loki_url="http://loki:3100",
            dry_run=False,
            stale_threshold_min=5,
            dead_threshold_min=30,
        )

        assert result["services"] == 0
        assert result["lines"] == 0
        # Should push a "(none)" placeholder entry
        mock_client.push_collection_status.assert_called_once()
        entries = mock_client.push_collection_status.call_args[0][0]
        assert entries[0]["service"] == "(none)"

    @patch("log_collection_status_reporter.LokiClient")
    def test_dry_run_no_push(self, mock_client_cls: MagicMock) -> None:
        mock_client = MagicMock()
        mock_client.list_services.return_value = ["svc-a"]
        mock_client.count_lines_by_service.return_value = {"svc-a": 42}
        mock_client.last_log_ns.return_value = int(time.time() * 1e9)
        mock_client_cls.return_value = mock_client

        result = run_reporter(
            loki_url="http://loki:3100",
            dry_run=True,
            stale_threshold_min=5,
            dead_threshold_min=30,
        )

        assert result["services"] == 1
        mock_client.push_collection_status.assert_not_called()

    @patch("log_collection_status_reporter.LokiClient")
    def test_sorting_dead_last(self, mock_client_cls: MagicMock) -> None:
        mock_client = MagicMock()
        mock_client.list_services.return_value = ["stale-svc", "healthy-svc", "dead-svc"]
        mock_client.count_lines_by_service.return_value = {"stale-svc": 10, "healthy-svc": 100, "dead-svc": 5}
        mock_client.last_log_ns.side_effect = [
            int(time.time() * 1e9) - int(10 * 60 * 1e9),  # stale
            int(time.time() * 1e9) - int(1 * 60 * 1e9),   # healthy
            int(time.time() * 1e9) - int(60 * 60 * 1e9),  # dead
        ]
        mock_client.push_collection_status.return_value = True
        mock_client_cls.return_value = mock_client

        run_reporter(
            loki_url="http://loki:3100",
            dry_run=False,
            stale_threshold_min=5,
            dead_threshold_min=30,
        )

        entries = mock_client.push_collection_status.call_args[0][0]
        statuses = [e["service_status"] for e in entries]
        # healthy first, then stale, then dead
        assert statuses == ["healthy", "stale", "dead"]
        # within same status, higher lines first
        assert entries[0]["service"] == "healthy-svc"
        assert entries[0]["total_lines"] == 100


if __name__ == "__main__":
    unittest.main()
