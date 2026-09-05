#!/usr/bin/env python3
"""Smoke tests for check_go_openapi_routes (no pytest required)."""

from __future__ import annotations

import importlib.util
import subprocess
import sys
from pathlib import Path

ROOT = Path(__file__).resolve().parents[3]
CHECKER = ROOT / "db" / "scripts" / "ci" / "check_go_openapi_routes.py"
CONFIG = ROOT / "db" / "scripts" / "ci" / "go_openapi_services.yaml"
LEGACY = ROOT / "db" / "scripts" / "ci" / "check_taskbill_openapi_routes.py"


def _load():
    spec = importlib.util.spec_from_file_location("check_go_openapi_routes", CHECKER)
    assert spec and spec.loader
    mod = importlib.util.module_from_spec(spec)
    # dataclasses on 3.14 requires the module to be registered before exec_module
    sys.modules[spec.name] = mod
    spec.loader.exec_module(mod)
    return mod


def run_checker(*extra: str) -> tuple[int, str]:
    completed = subprocess.run(
        [sys.executable, str(CHECKER), *extra],
        cwd=ROOT,
        capture_output=True,
        text=True,
        check=False,
    )
    return completed.returncode, (completed.stdout or "") + (completed.stderr or "")


def test_all_services_pass_fail_on_extra() -> None:
    code, out = run_checker("--fail-on-extra", "--no-warn-extra")
    assert code == 0, out
    assert "OK [taskAuth]" in out, out
    assert "OK [taskProjectService]" in out, out
    assert "OK [taskCloudService]" in out, out
    assert "OK [taskTaskService]" in out, out
    assert "OK [taskAIComment]" in out, out
    assert "OK [taskBill]" in out, out


def test_legacy_taskbill_shim() -> None:
    completed = subprocess.run(
        [sys.executable, str(LEGACY), "--fail-on-extra"],
        cwd=ROOT,
        capture_output=True,
        text=True,
        check=False,
    )
    out = (completed.stdout or "") + (completed.stderr or "")
    assert completed.returncode == 0, out
    assert "OK [taskBill]" in out, out


def test_stale_fails_with_fail_on_extra() -> None:
    mod = _load()
    _, services = mod.load_config(CONFIG)
    spec = services["taskProjectService"]
    import yaml

    handlers = spec.handlers.read_text(encoding="utf-8")
    doc = yaml.safe_load(spec.openapi.read_text(encoding="utf-8"))
    # 须落在 prefix_routers 之外，否则会被 /api/tenant/ 前缀路由器盖住
    doc["paths"]["/api/__stale_probe__/"] = {
        "get": {"summary": "probe"}
    }
    code, report = mod.run_check_service(
        spec, handlers, doc, warn_extra=True, fail_on_extra=True
    )
    assert code == 1, report
    assert "__stale_probe__" in report, report


def test_missing_mounted_fails() -> None:
    mod = _load()
    _, services = mod.load_config(CONFIG)
    spec = services["taskAuth"]
    import yaml

    handlers = spec.handlers.read_text(encoding="utf-8")
    doc = yaml.safe_load(spec.openapi.read_text(encoding="utf-8"))
    del doc["paths"]["/api/health/"]
    internal = yaml.safe_load(spec.openapi_internal.read_text(encoding="utf-8"))
    code, report = mod.run_check_service(
        spec,
        handlers,
        doc,
        warn_extra=False,
        fail_on_extra=False,
        openapi_internal_doc=internal,
    )
    assert code == 1, report
    assert "health" in report, report


def test_auth_internal_covers_skipped_mounts() -> None:
    mod = _load()
    _, services = mod.load_config(CONFIG)
    spec = services["taskAuth"]
    import yaml

    handlers = spec.handlers.read_text(encoding="utf-8")
    doc = yaml.safe_load(spec.openapi.read_text(encoding="utf-8"))
    internal = yaml.safe_load(spec.openapi_internal.read_text(encoding="utf-8"))
    del internal["paths"]["/api/internal/verification-code/verify/"]
    code, report = mod.run_check_service(
        spec,
        handlers,
        doc,
        warn_extra=False,
        fail_on_extra=True,
        openapi_internal_doc=internal,
    )
    assert code == 1, report
    assert "verification-code" in report, report


def test_cloud_internal_complete_covers_all_skips() -> None:
    mod = _load()
    _, services = mod.load_config(CONFIG)
    spec = services["taskCloudService"]
    import yaml

    assert spec.internal_coverage == "complete"
    handlers = spec.handlers.read_text(encoding="utf-8")
    doc = yaml.safe_load(spec.openapi.read_text(encoding="utf-8"))
    internal = yaml.safe_load(spec.openapi_internal.read_text(encoding="utf-8"))
    code, report = mod.run_check_service(
        spec,
        handlers,
        doc,
        warn_extra=False,
        fail_on_extra=True,
        openapi_internal_doc=internal,
    )
    assert code == 0, report
    assert not any(
        str(p).startswith("/api/internal") for p in (doc.get("paths") or {})
    ), report
    # drop one documented internal → must fail under complete coverage
    del internal["paths"][next(iter(internal["paths"]))]
    code2, report2 = mod.run_check_service(
        spec,
        handlers,
        doc,
        warn_extra=False,
        fail_on_extra=True,
        openapi_internal_doc=internal,
    )
    assert code2 == 1, report2


def test_taskbill_internal_split() -> None:
    mod = _load()
    _, services = mod.load_config(CONFIG)
    spec = services["taskBill"]
    import yaml

    assert spec.openapi_internal is not None
    handlers = spec.handlers.read_text(encoding="utf-8")
    doc = yaml.safe_load(spec.openapi.read_text(encoding="utf-8"))
    internal = yaml.safe_load(spec.openapi_internal.read_text(encoding="utf-8"))
    assert not any(
        str(p).startswith("/api/internal") for p in (doc.get("paths") or {})
    )
    assert any(
        str(p).startswith("/api/internal/taskbill/") for p in (internal.get("paths") or {})
    )
    code, report = mod.run_check_service(
        spec,
        handlers,
        doc,
        warn_extra=False,
        fail_on_extra=True,
        openapi_internal_doc=internal,
    )
    assert code == 0, report


def test_cli_single_service() -> None:
    code, out = run_checker("--service", "taskAuth", "--fail-on-extra")
    assert code == 0, out
    assert "OK [taskAuth]" in out and "taskCloudService" not in out, out


def main() -> int:
    tests = [
        test_all_services_pass_fail_on_extra,
        test_legacy_taskbill_shim,
        test_stale_fails_with_fail_on_extra,
        test_missing_mounted_fails,
        test_auth_internal_covers_skipped_mounts,
        test_cloud_internal_complete_covers_all_skips,
        test_taskbill_internal_split,
        test_cli_single_service,
    ]
    failed = 0
    for fn in tests:
        try:
            fn()
            print(f"PASS {fn.__name__}")
        except Exception as exc:  # noqa: BLE001
            failed += 1
            print(f"FAIL {fn.__name__}: {exc}")
    if failed:
        print(f"{failed}/{len(tests)} failed")
        return 1
    print(f"OK: {len(tests)} tests passed")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
