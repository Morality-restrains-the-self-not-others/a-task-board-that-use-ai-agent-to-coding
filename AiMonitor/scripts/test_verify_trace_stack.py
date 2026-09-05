"""Unit tests for verify_trace_stack constants."""

from __future__ import annotations

from verify_trace_stack import SMOKE_SPAN_HEX, SMOKE_TRACE_HEX


def test_smoke_trace_id_is_valid_hex32() -> None:
    assert len(SMOKE_TRACE_HEX) == 32
    int(SMOKE_TRACE_HEX, 16)


def test_smoke_span_id_is_valid_hex16() -> None:
    assert len(SMOKE_SPAN_HEX) == 16
    int(SMOKE_SPAN_HEX, 16)
