#!/usr/bin/env python3
"""Regression: GET /api/auth/verify must not share taskauth-login limit-req.

Bug family: /api/auth/* catch-all (taskauth-login, auth_mode public,
limit-req 0.5/s) inherited every /api/auth/ route not explicitly lifted.
user-roles/user-permissions were fixed in OPT-20260824; /api/auth/verify is a
logged-in session read (Go microservice fallback auth) that can be fired in
parallel and must sit on a token route with priority > taskauth-login.
"""

from __future__ import annotations

import sys
from pathlib import Path

import yaml

REPO = Path(__file__).resolve().parents[3]
ROUTES = REPO / "taskGateway" / "routes" / "routes.yaml"

REQUIRED_URIS = (
    "/api/auth/verify",
)


def _load_routes():
    doc = yaml.safe_load(ROUTES.read_text(encoding="utf-8"))
    return doc.get("routes") or []


def _route_covers(route: dict, uri: str) -> bool:
    uris = list(route.get("uris") or [])
    if route.get("uri"):
        uris.append(route["uri"])
    return uri in uris


def _check():
    routes = _load_routes()
    login = next((r for r in routes if r.get("id") == "taskauth-login"), None)
    assert login is not None, "taskauth-login route missing"
    login_prio = int(login.get("priority") or 0)

    for uri in REQUIRED_URIS:
        winners = [
            r
            for r in routes
            if r.get("id") != "taskauth-login" and _route_covers(r, uri)
        ]
        assert winners, (
            f"{uri} is not listed on any route other than taskauth-login; "
            "it inherits login limit-req (0.5/s) and stalls ~2s"
        )
        best = max(winners, key=lambda r: int(r.get("priority") or 0))
        assert int(best.get("priority") or 0) > login_prio, (
            f"{uri} owner {best.get('id')} priority {best.get('priority')} "
            f"must be > taskauth-login {login_prio}"
        )
        assert best.get("auth_mode") == "token", (
            f"{uri} owner {best.get('id')} auth_mode={best.get('auth_mode')!r} want token "
            "(authenticated session API, not login brute-force limiter)"
        )


def test_auth_verify_not_on_login_ratelimit():
    """GET /api/auth/verify 不得落入 login 0.5/s 限流。"""
    _check()


def main() -> int:
    try:
        _check()
    except AssertionError as e:
        print(f"FAIL: {e}", file=sys.stderr)
        return 1
    print("PASS: /api/auth/verify beats taskauth-login")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
