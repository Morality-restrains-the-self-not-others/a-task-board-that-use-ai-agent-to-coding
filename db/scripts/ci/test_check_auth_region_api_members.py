#!/usr/bin/env python3
"""Self-test for check_auth_region_api_members (no pytest required)."""

from __future__ import annotations

import importlib.util
import subprocess
import sys
import tempfile
from pathlib import Path

ROOT = Path(__file__).resolve().parents[3]
CHECKER = ROOT / "db" / "scripts" / "ci" / "check_auth_region_api_members.py"


def _load():
    spec = importlib.util.spec_from_file_location("check_auth_region_api_members", CHECKER)
    assert spec and spec.loader
    mod = importlib.util.module_from_spec(spec)
    sys.modules[spec.name] = mod
    spec.loader.exec_module(mod)
    return mod


def test_repo_current_seeds_pass() -> None:
    completed = subprocess.run(
        [sys.executable, str(CHECKER), "--root", str(ROOT)],
        cwd=ROOT,
        capture_output=True,
        text=True,
        check=False,
    )
    out = (completed.stdout or "") + (completed.stderr or "")
    assert completed.returncode == 0, out
    assert "OK auth region api members" in out, out


def test_parse_and_format_validation() -> None:
    mod = _load()
    sql = """
INSERT IGNORE INTO auth_resource_member (id, resource_group_id, member_kind, member_key) VALUES
('a', 'rg', 'api', 'GET /api/auth/resource-groups/'),
('b', 'rg', 'ui', 'PeopleAccess.SubjectList'),
('c', 'rg', 'api', 'PUT /api/auth/roles/role_id/{rid}/resource-groups/');
"""
    members = mod.parse_api_members_from_sql(sql, "fixture.sql")
    assert len(members) == 2
    assert members[0].method == "GET"
    assert mod.validate_member_key_format("GET /api/auth/resource-groups/") is None
    assert mod.validate_member_key_format("FETCH /api/x") is not None
    assert mod.validate_member_key_format("GET /v1/nope") is not None


def test_path_template_prefix_always_trailing_slash() -> None:
    mod = _load()
    assert mod._path_template_prefix("/api/tenant/current") == "/api/tenant/current/"
    assert mod._path_template_prefix("/api/tenant/current/") == "/api/tenant/current/"
    assert mod._path_template_prefix("/api/x/{id}/") == "/api/x/{}/"


def test_route_covers_prefix_and_catchall() -> None:
    mod = _load()
    member = mod.ApiMember(
        key="PUT /api/tenant/member-role/",
        method="PUT",
        path="/api/tenant/member-role/",
        source="t",
    )
    routes = {
        "ANY /api/tenant/",
        'PUT /api/tenant/member-role/company_id/{cid}/member_id/{mid}/',
    }
    # normalize routes via extract would already normalize; use as stored
    routes = {f"{a} {mod._norm_path(b)}" for a, b in (r.split(" ", 1) for r in routes)}
    assert mod.route_covers(member, routes)


def test_missing_member_fails_fixture_repo() -> None:
    mod = _load()
    with tempfile.TemporaryDirectory() as tmp:
        root = Path(tmp)
        (root / "db").mkdir()
        (root / "db" / "registry.yaml").write_text("databases: {}\n", encoding="utf-8")
        migrate = root / "dataMigrate" / "taskAuth"
        migrate.mkdir(parents=True)
        (migrate / "001_seed.sql").write_text(
            """
INSERT IGNORE INTO auth_resource_member (id, resource_group_id, member_kind, member_key) VALUES
('rm-x', 'rg', 'api', 'GET /api/auth/definitely-missing-route/');
""",
            encoding="utf-8",
        )
        go_dir = root / "taskAuth" / "src"
        go_dir.mkdir(parents=True)
        (go_dir / "handlers.go").write_text(
            'package main\nfunc mount(){ mux.HandleFunc("GET /api/health/", h) }\n',
            encoding="utf-8",
        )
        code, report = mod.run_check(root)
        assert code == 1, report
        assert "definitely-missing-route" in report, report


def test_matching_handlefunc_passes_fixture() -> None:
    mod = _load()
    with tempfile.TemporaryDirectory() as tmp:
        root = Path(tmp)
        (root / "db").mkdir()
        (root / "db" / "registry.yaml").write_text("databases: {}\n", encoding="utf-8")
        migrate = root / "dataMigrate" / "taskAuth"
        migrate.mkdir(parents=True)
        (migrate / "001_seed.sql").write_text(
            """
INSERT IGNORE INTO auth_resource_member (id, resource_group_id, member_kind, member_key) VALUES
('rm-x', 'rg', 'api', 'GET /api/auth/resource-groups/');
""",
            encoding="utf-8",
        )
        go_dir = root / "taskAuth" / "src"
        go_dir.mkdir(parents=True)
        (go_dir / "handlers.go").write_text(
            'package main\nfunc mount(){ mux.HandleFunc("GET /api/auth/resource-groups/", h) }\n',
            encoding="utf-8",
        )
        code, report = mod.run_check(root)
        assert code == 0, report


def main() -> int:
    tests = [
        test_parse_and_format_validation,
        test_path_template_prefix_always_trailing_slash,
        test_route_covers_prefix_and_catchall,
        test_missing_member_fails_fixture_repo,
        test_matching_handlefunc_passes_fixture,
        test_repo_current_seeds_pass,
    ]
    failed = 0
    for fn in tests:
        try:
            fn()
            print(f"PASS {fn.__name__}")
        except Exception as e:  # noqa: BLE001 — selftest reporter
            failed += 1
            print(f"FAIL {fn.__name__}: {e}", file=sys.stderr)
    if failed:
        print(f"{failed}/{len(tests)} failed", file=sys.stderr)
        return 1
    print(f"OK {len(tests)} tests")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
