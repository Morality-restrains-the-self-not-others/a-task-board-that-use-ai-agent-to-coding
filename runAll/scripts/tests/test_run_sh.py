"""runAll/run.sh — setsid nohup 独立会话，避免 screen 前台作业被 SIGTERM 带走。"""

from __future__ import annotations

import os
import stat
import subprocess
import time
from pathlib import Path

import pytest

RUN_SH = Path(__file__).resolve().parents[2] / "run.sh"

STUB = r"""#!/usr/bin/env python3
import os, sys, time
from pathlib import Path
out = Path(os.environ["RUNALL_STUB_OUT"])
out.write_text(
    "pid={pid}\nppid={ppid}\npgid={pgid}\nsid={sid}\nargv={argv}\n".format(
        pid=os.getpid(),
        ppid=os.getppid(),
        pgid=os.getpgid(0),
        sid=os.getsid(0),
        argv=" ".join(sys.argv[1:]),
    ),
    encoding="utf-8",
)
if os.environ.get("RUNALL_STUB_SLEEP") == "1":
    time.sleep(30)
"""


def _parse_coords(text: str) -> dict[str, str]:
    kv = {}
    for line in text.splitlines():
        if "=" in line:
            k, v = line.split("=", 1)
            kv[k] = v
    return kv


def _wait_stub(out: Path, timeout: float = 3.0) -> str:
    deadline = time.time() + timeout
    while time.time() < deadline:
        if out.is_file() and out.stat().st_size > 0:
            return out.read_text(encoding="utf-8")
        time.sleep(0.05)
    raise AssertionError(f"stub did not write {out}")


def _kill_stub(coords: dict[str, str]) -> None:
    pid = int(coords["pid"])
    pgid = int(coords["pgid"])
    try:
        os.killpg(pgid, 9)
    except OSError:
        try:
            os.kill(pid, 9)
        except OSError:
            pass


@pytest.fixture
def stub_env(tmp_path: Path):
    stub = tmp_path / "fake-runAll"
    stub.write_text(STUB, encoding="utf-8")
    stub.chmod(stub.stat().st_mode | stat.S_IEXEC)
    out = tmp_path / "coords.txt"
    log = tmp_path / "console.log"
    env = os.environ.copy()
    env["RUNALL_SKIP_BUILD"] = "1"
    env["RUNALL_BIN"] = str(stub)
    env["RUNALL_CONSOLE_LOG"] = str(log)
    env["RUNALL_CONFIG"] = str(tmp_path / "unused.yaml")
    env["RUNALL_STUB_OUT"] = str(out)
    env.pop("RUNALL_FOREGROUND", None)
    return env, out, log


def test_run_sh_daemonizes_new_session(stub_env):
    env, out, log = stub_env
    env["RUNALL_STUB_SLEEP"] = "1"
    started = time.monotonic()
    proc = subprocess.run(
        ["bash", str(RUN_SH)],
        env=env,
        capture_output=True,
        text=True,
        timeout=8,
        check=False,
    )
    elapsed = time.monotonic() - started
    assert proc.returncode == 0, proc.stderr + proc.stdout
    assert elapsed < 4, f"run.sh blocked for {elapsed:.1f}s; expected immediate return"
    assert "setsid nohup" in proc.stdout
    coords = _parse_coords(_wait_stub(out))
    try:
        pid, pgid, sid = int(coords["pid"]), int(coords["pgid"]), int(coords["sid"])
        assert pid == pgid == sid, f"expected session leader, got {coords}"
        assert "--config" in coords["argv"]
        assert log.is_file()
    finally:
        _kill_stub(coords)


def test_run_sh_foreground_for_cli_command(stub_env):
    env, out, log = stub_env
    proc = subprocess.run(
        ["bash", str(RUN_SH), "-command", "doctor"],
        env=env,
        capture_output=True,
        text=True,
        timeout=8,
        check=False,
    )
    assert proc.returncode == 0, proc.stderr + proc.stdout
    assert "foreground" in proc.stdout
    coords = _parse_coords(_wait_stub(out))
    assert "-command" in coords["argv"] and "doctor" in coords["argv"]
    assert not log.is_file() or log.stat().st_size == 0
