"""Edge gateway route value objects (configuration domain, not APISIX runtime)."""

from __future__ import annotations

from dataclasses import dataclass
from typing import Literal

AuthMode = Literal["deny", "none", "app-jwt", "token"]


@dataclass(frozen=True)
class UpstreamRef:
    """Logical upstream service name (maps to conf/task-gateway upstreams)."""

    service: str  # django | taskAuth | gitOauth | taskSse | taskContainerGateway


@dataclass(frozen=True)
class GatewayRoute:
    """Single route rule in the gateway route table."""

    id: str
    uri_pattern: str
    upstream: UpstreamRef
    auth_mode: AuthMode
    priority: int = 0
    methods: tuple[str, ...] = ()


@dataclass(frozen=True)
class GatewayRouteTable:
    """Aggregate root: routes published together."""

    version: str
    routes: tuple[GatewayRoute, ...]
