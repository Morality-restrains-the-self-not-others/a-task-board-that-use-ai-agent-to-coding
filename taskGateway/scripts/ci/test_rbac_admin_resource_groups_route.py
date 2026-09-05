#!/usr/bin/env python3
"""Regression: /api/auth/resource-groups/ must be under taskauth-rbac-admin (token).

Bug: catalog fell under taskauth-login (auth_mode: public) → no X-Tenant-Perms →
handleListResourceGroups always 403 → FE showed misleading dataMigrate 032 text.

夜间 pytest 抽测曾报 "no tests ran"：本文件只有 main()，pytest 收集不到 test_*。
重构为 pytest 可收集的 test_rbac_admin_resource_groups_route()，保留 main() 直接运行路径。
"""

from __future__ import annotations

import sys
from pathlib import Path

import yaml

REPO = Path(__file__).resolve().parents[3]
ROUTES = REPO / "taskGateway" / "routes" / "routes.yaml"

REQUIRED_URIS = (
    "/api/auth/resource-groups/",
    "/api/auth/resource-groups/*",
)


def _load_routes():
    doc = yaml.safe_load(ROUTES.read_text(encoding="utf-8"))
    return doc.get("routes") or []


def _check():
    """Perform all assertions. Raise AssertionError with detail on failure."""
    routes = _load_routes()
    rbac = next((r for r in routes if r.get("id") == "taskauth-rbac-admin"), None)
    assert rbac is not None, "taskauth-rbac-admin route missing"
    assert rbac.get("auth_mode") == "token", (
        f"taskauth-rbac-admin auth_mode={rbac.get('auth_mode')!r} want token"
    )
    uris = set(rbac.get("uris") or [])
    missing = [u for u in REQUIRED_URIS if u not in uris]
    assert not missing, f"resource-groups uris missing from taskauth-rbac-admin: {missing}"

    # Ensure public login catch-all cannot win for catalog path when both match:
    # rbac-admin priority must be higher than taskauth-login.
    login = next((r for r in routes if r.get("id") == "taskauth-login"), None)
    if login is not None:
        assert int(rbac.get("priority") or 0) > int(login.get("priority") or 0), (
            f"taskauth-rbac-admin priority {rbac.get('priority')} "
            f"must be > taskauth-login {login.get('priority')}"
        )


def test_rbac_admin_resource_groups_route():
    """/api/auth/resource-groups/ 必须由 taskauth-rbac-admin (token) 承接。"""
    _check()


def main() -> int:
    try:
        _check()
    except AssertionError as e:
        print(f"FAIL: {e}", file=sys.stderr)
        return 1
    print("PASS: /api/auth/resource-groups/ is under taskauth-rbac-admin (token)")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
