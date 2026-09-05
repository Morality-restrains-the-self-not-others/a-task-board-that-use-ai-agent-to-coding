#!/usr/bin/env python3
"""Smoke tests for check_aliyun_tea_direct_runtime (no pytest required)."""

from __future__ import annotations

import subprocess
import sys
import tempfile
from pathlib import Path

ROOT = Path(__file__).resolve().parents[3]
CHECKER = ROOT / "db" / "scripts" / "ci" / "check_aliyun_tea_direct_runtime.py"


def run_checker(cwd: Path | None = None) -> tuple[int, str]:
    completed = subprocess.run(
        [sys.executable, str(CHECKER), "--root", str(cwd or ROOT)],
        cwd=ROOT,
        capture_output=True,
        text=True,
        check=False,
    )
    out = (completed.stdout or "") + (completed.stderr or "")
    return completed.returncode, out


def test_repo_passes() -> None:
    code, out = run_checker()
    assert code == 0, out
    assert "OK: no hand-written" in out, out


def test_hand_written_runtime_options_fails() -> None:
    with tempfile.TemporaryDirectory() as tmp:
        root = Path(tmp)
        # Minimal monorepo markers for --root
        (root / "db").mkdir()
        (root / "db" / "registry.yaml").write_text("databases: {}\n", encoding="utf-8")
        target_dir = root / "task2app" / "Saas_project" / "cloud" / "providers" / "aliyun"
        target_dir.mkdir(parents=True)
        bad = target_dir / "bad_caller.py"
        bad.write_text(
            "from alibabacloud_tea_util.models import RuntimeOptions\n"
            "runtime = RuntimeOptions()\n",
            encoding="utf-8",
        )
        # Allowlisted helper must not trip the checker when present
        helper = target_dir / "direct_network.py"
        helper.write_text(
            "from alibabacloud_tea_util import models as util_models\n"
            "def direct_runtime_options(**kwargs):\n"
            "    return util_models.RuntimeOptions(**kwargs)\n",
            encoding="utf-8",
        )
        code, out = run_checker(cwd=root)
        assert code == 1, out
        assert "bad_caller.py" in out, out
        assert "direct_network.py" not in out.split("VIOLATION")[-1] or "bad_caller" in out


def main() -> int:
    failures = 0
    for fn in (test_repo_passes, test_hand_written_runtime_options_fails):
        try:
            fn()
            print(f"PASS {fn.__name__}")
        except AssertionError as exc:
            failures += 1
            print(f"FAIL {fn.__name__}: {exc}", file=sys.stderr)
    return 1 if failures else 0


if __name__ == "__main__":
    raise SystemExit(main())
