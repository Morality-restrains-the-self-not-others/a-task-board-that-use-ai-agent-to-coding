"""runAll/run.sh clone-run must source cutover.env; it is a required dependency."""

from __future__ import annotations

import os
import subprocess
from pathlib import Path

import pytest

RUN_SH = Path(__file__).resolve().parents[2] / "run.sh"
DUMP_KEYS = (
    "DEPLOY_MODE",
    "DEPLOY_ROOT",
    "RUNALL_SKIP_BUILD",
    "RUNALL_BIN",
    "CONF_ROOT",
)


def _write_dumped_run_sh(root: Path) -> Path:
    (root / "runAll").mkdir()
    src = RUN_SH.read_text(encoding="utf-8")
    marker = 'if [[ "${RUNALL_SKIP_BUILD:-0}" != "1" ]]; then'
    assert marker in src
    inject = (
        'printf "DEPLOY_MODE=%s\\nRUNALL_SKIP_BUILD=%s\\nDEPLOY_ROOT=%s\\n'
        'RUNALL_BIN=%s\\nCONF_ROOT=%s\\n" '
        '"${DEPLOY_MODE:-}" "${RUNALL_SKIP_BUILD:-}" "${DEPLOY_ROOT:-}" '
        '"${RUNALL_BIN:-}" "${CONF_ROOT:-}"\n'
        "exit 0\n" + marker
    )
    dest = root / "runAll" / "run.sh"
    dest.write_text(src.replace(marker, inject, 1), encoding="utf-8")
    dest.chmod(0o755)
    return dest


def _bin_runall(root: Path) -> None:
    (root / "bin").mkdir()
    fake = root / "bin" / "runAll"
    fake.write_text("#!/bin/sh\nexit 0\n", encoding="utf-8")
    fake.chmod(0o755)


def _run_dump(dest: Path, root: Path) -> subprocess.CompletedProcess[str]:
    env = os.environ.copy()
    for k in DUMP_KEYS:
        env.pop(k, None)
    env.pop("CUTOVER_ENV", None)
    return subprocess.run(
        ["bash", str(dest)],
        cwd=root / "runAll",
        capture_output=True,
        text=True,
        check=False,
        env=env,
    )


def _parse_dump(stdout: str) -> dict[str, str]:
    got: dict[str, str] = {}
    for line in stdout.splitlines():
        if "=" in line:
            k, _, v = line.partition("=")
            if k in DUMP_KEYS:
                got[k] = v
    return got


def test_run_sh_clone_run_requires_cutover_env(tmp_path: Path) -> None:
    _bin_runall(tmp_path)
    dest = _write_dumped_run_sh(tmp_path)
    proc = _run_dump(dest, tmp_path)
    assert proc.returncode != 0, proc.stdout
    assert "cutover.env" in proc.stderr
    assert "up.sh" in proc.stderr


def test_run_sh_clone_run_sources_cutover_env(tmp_path: Path) -> None:
    _bin_runall(tmp_path)
    cutover = tmp_path / "cutover.env"
    cutover.write_text(
        "\n".join(
            [
                f"export DEPLOY_ROOT={tmp_path}",
                f"export CONF_ROOT={tmp_path}/conf",
                "export DEPLOY_MODE=1",
                "export RUNALL_SKIP_BUILD=1",
                f"export RUNALL_BIN={tmp_path}/bin/runAll",
            ]
        )
        + "\n",
        encoding="utf-8",
    )
    dest = _write_dumped_run_sh(tmp_path)
    proc = _run_dump(dest, tmp_path)
    if proc.returncode != 0:
        pytest.fail(
            f"run.sh dump failed rc={proc.returncode} stderr={proc.stderr!r} stdout={proc.stdout!r}"
        )
    got = _parse_dump(proc.stdout)
    assert got.get("DEPLOY_MODE") == "1"
    assert got.get("RUNALL_SKIP_BUILD") == "1"
    assert got.get("DEPLOY_ROOT") == str(tmp_path)
    assert got.get("RUNALL_BIN") == str(tmp_path / "bin" / "runAll")
    assert got.get("CONF_ROOT") == str(tmp_path / "conf")


def test_run_sh_does_not_force_deploy_mode_in_source_tree(tmp_path: Path) -> None:
    _bin_runall(tmp_path)
    (tmp_path / "taskAuth").mkdir()
    dest = _write_dumped_run_sh(tmp_path)
    proc = _run_dump(dest, tmp_path)
    if proc.returncode != 0:
        pytest.fail(
            f"run.sh dump failed rc={proc.returncode} stderr={proc.stderr!r} stdout={proc.stdout!r}"
        )
    got = _parse_dump(proc.stdout)
    assert got.get("DEPLOY_MODE", "") == ""
    assert got.get("RUNALL_SKIP_BUILD", "") == ""
