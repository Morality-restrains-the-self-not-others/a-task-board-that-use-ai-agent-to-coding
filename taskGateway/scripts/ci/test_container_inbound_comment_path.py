#!/usr/bin/env python3
"""Container inbound gateway routes must require /comment/{cid}/."""

from __future__ import annotations

from pathlib import Path

import yaml

REPO = Path(__file__).resolve().parents[3]
ROUTES = REPO / "taskGateway" / "routes" / "routes.yaml"

FORBIDDEN = "/task/*/cloud/"
REQUIRED = "/task/*/comment/*/cloud/"


def test_container_inbound_token_requires_comment_segment() -> None:
    doc = yaml.safe_load(ROUTES.read_text(encoding="utf-8"))
    route = next((r for r in doc.get("routes") or [] if r.get("id") == "container-inbound-token"), None)
    assert route is not None, "container-inbound-token route missing"
    uris = list(route.get("uris") or [])
    assert uris, "container-inbound-token has no uris"
    for uri in uris:
        assert REQUIRED in uri, f"uri missing comment segment: {uri}"
        assert FORBIDDEN not in uri, f"legacy uri still routed: {uri}"


if __name__ == "__main__":
    test_container_inbound_token_requires_comment_segment()
    print("ok")
