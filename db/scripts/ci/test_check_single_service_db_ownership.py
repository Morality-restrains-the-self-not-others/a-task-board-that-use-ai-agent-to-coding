#!/usr/bin/env python3
"""Smoke tests for check_single_service_db_ownership (no pytest required)."""

from __future__ import annotations

import subprocess
import sys
from pathlib import Path

ROOT = Path(__file__).resolve().parents[3]
CHECKER = ROOT / "db" / "scripts" / "ci" / "check_single_service_db_ownership.py"


def run_checker() -> tuple[int, str]:
    completed = subprocess.run(
        [sys.executable, str(CHECKER)],
        cwd=ROOT,
        capture_output=True,
        text=True,
        check=False,
    )
    out = (completed.stdout or "") + (completed.stderr or "")
    return completed.returncode, out


def test_repo_passes_with_registered_known_debt_warnings() -> None:
    """Checker exits 0; registered known_cross_service_access appears as WARN count."""
    code, out = run_checker()
    assert code == 0, out
    assert "OK: single-service DB ownership" in out, out
    # Count comes from known_cross_service_access in table_ownership.yaml
    # （saas 已退役（OPT-052），taskCloudService saas_db.go 死代码已删 → 0 告警为当前正确态）
    assert "known-debt warnings" in out, out


def test_unregistered_cross_access_fails() -> None:
    """Inject a fake non-owner reference and expect failure."""
    probe = ROOT / "taskAuth" / "src" / "_ownership_ci_probe_tmp.go"
    assert not probe.exists(), "probe file already exists"
    try:
        probe.write_text(
            '// temporary CI probe\nconst x = "db/saas/saas.sqlite3"\n',
            encoding="utf-8",
        )
        code, out = run_checker()
        assert code == 1, out
        assert "VIOLATION" in out and "task-auth" in out and "saas" in out, out
    finally:
        if probe.exists():
            probe.unlink()


def main() -> int:
    failures = 0
    for fn in (
        test_repo_passes_with_registered_known_debt_warnings,
        test_unregistered_cross_access_fails,
    ):
        try:
            fn()
            print(f"PASS {fn.__name__}")
        except AssertionError as exc:
            failures += 1
            print(f"FAIL {fn.__name__}: {exc}", file=sys.stderr)
    return 1 if failures else 0


if __name__ == "__main__":
    raise SystemExit(main())
