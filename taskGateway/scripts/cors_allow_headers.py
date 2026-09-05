"""Canonical CORS allow-headers list for taskGateway (APISIX + platform Go services)."""

from __future__ import annotations

# Single source of truth — consumed by routes-to-apisix.py and gatewaycors Go codegen.
CORS_ALLOW_HEADERS: tuple[str, ...] = (
    "Authorization",
    "Content-Type",
    "X-Trace-Id",
    "X-Span-Id",
    "X-Parent-Span-Id",
    "traceparent",
    "X-CSRFToken",
    "X-Requested-With",
    "Idempotency-Key",
)

# Response headers that browsers may read via JS (Access-Control-Expose-Headers).
# Exposing X-Trace-Id enables frontend data-traceId binding for error diagnostics.
CORS_EXPOSE_HEADERS: tuple[str, ...] = (
    "X-Trace-Id",
    "X-Parent-Span-Id",
    "traceparent",
)


def cors_allow_headers_apisix() -> str:
    """Comma-separated header list for APISIX cors plugin (no spaces)."""
    return ",".join(CORS_ALLOW_HEADERS)


def cors_expose_headers_apisix() -> str:
    """Comma-separated expose-headers list for APISIX cors plugin (no spaces)."""
    return ",".join(CORS_EXPOSE_HEADERS)


def cors_allow_headers_go(extra: list[str] | None = None) -> str:
    """Human-readable header list for Go Access-Control-Allow-Headers."""
    headers = list(CORS_ALLOW_HEADERS)
    for name in extra or []:
        if name and name not in headers:
            headers.append(name)
    return ", ".join(headers)


def cors_expose_headers_go() -> str:
    """Human-readable header list for Go Access-Control-Expose-Headers."""
    return ", ".join(CORS_EXPOSE_HEADERS)
