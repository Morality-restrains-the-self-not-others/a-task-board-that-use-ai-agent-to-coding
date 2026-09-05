#!/usr/bin/env python3
"""Smoke tests for check_cloud_sdk_direct_network."""

from __future__ import annotations

import subprocess
import sys
import tempfile
from pathlib import Path

ROOT = Path(__file__).resolve().parents[3]
CHECKER = ROOT / "db" / "scripts" / "ci" / "check_cloud_sdk_direct_network.py"


def run_checker(cwd: Path | None = None) -> tuple[int, str]:
    completed = subprocess.run(
        [sys.executable, str(CHECKER), "--root", str(cwd or ROOT)],
        cwd=ROOT,
        capture_output=True,
        text=True,
        check=False,
    )
    return completed.returncode, (completed.stdout or "") + (completed.stderr or "")


def test_repo_passes() -> None:
    code, out = run_checker()
    assert code == 0, out
    assert "OK:" in out, out


def test_boto3_without_helper_fails() -> None:
    with tempfile.TemporaryDirectory() as tmp:
        root = Path(tmp)
        (root / "db").mkdir()
        (root / "db" / "registry.yaml").write_text("databases: {}\n", encoding="utf-8")
        target = root / "task2app" / "Saas_project" / "cloud"
        target.mkdir(parents=True)
        (target / "bad_aws.py").write_text(
            "import boto3\nclient = boto3.client('s3')\n",
            encoding="utf-8",
        )
        code, out = run_checker(cwd=root)
        assert code == 1, out
        assert "bad_aws.py" in out and "aws-boto3" in out, out


def main() -> int:
    failures = 0
    for fn in (test_repo_passes, test_boto3_without_helper_fails):
        try:
            fn()
            print(f"PASS {fn.__name__}")
        except AssertionError as exc:
            failures += 1
            print(f"FAIL {fn.__name__}: {exc}", file=sys.stderr)
    return 1 if failures else 0


if __name__ == "__main__":
    raise SystemExit(main())
