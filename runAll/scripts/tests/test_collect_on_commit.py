"""collect-deploy-binaries-on-commit.sh is fail-open and skips when no sources."""

from __future__ import annotations

import os
import subprocess
from pathlib import Path

HOOK = Path(__file__).resolve().parents[3] / "scripts" / "lib" / "collect-deploy-binaries-on-commit.sh"
COLLECT = Path(__file__).resolve().parents[1] / "collect-deploy-binaries.sh"


def _write(p: Path, text: str = "ok\n") -> None:
    p.parent.mkdir(parents=True, exist_ok=True)
    p.write_text(text, encoding="utf-8")


def _hook_env(meta: Path, **extra: str) -> dict[str, str]:
    drop = {
        "COLLECT_SRC",
        "COLLECT_STAGING",
        "COLLECT_SOFT",
        "RAM_DEPLOY",
        "META_ROOT",
        "SESSION_META_ROOT",
        "SKIP_COLLECT_DEPLOY_BINARIES",
        "COLLECT_DEPLOY_BINARIES_REQUIRED",
        "CI",
    }
    env = {k: v for k, v in os.environ.items() if k not in drop}
    env["SESSION_META_ROOT"] = str(meta)
    env["COLLECT_STAGING"] = str(meta / "no-staging")
    env["RAM_DEPLOY"] = str(meta / "no-ram")
    env.update(extra)
    return env


def _fake_meta(tmp_path: Path) -> Path:
    meta = tmp_path / "meta"
    _write(meta / ".gitmodules", "[submodule \"runAll\"]\n\tpath = runAll\n")
    dest = meta / "runAll" / "scripts" / "collect-deploy-binaries.sh"
    dest.parent.mkdir(parents=True, exist_ok=True)
    dest.write_bytes(COLLECT.read_bytes())
    dest.chmod(0o755)
    return meta


def test_hook_skips_when_flag_set(tmp_path):
    meta = _fake_meta(tmp_path)
    r = subprocess.run(
        ["bash", str(HOOK)],
        env=_hook_env(meta, SKIP_COLLECT_DEPLOY_BINARIES="1"),
        capture_output=True,
        text=True,
        check=False,
    )
    assert r.returncode == 0, r.stderr + r.stdout
    assert not (meta / "deploy-binaries").exists()


def test_hook_skips_without_runall_source(tmp_path):
    meta = _fake_meta(tmp_path)
    r = subprocess.run(
        ["bash", str(HOOK)],
        env=_hook_env(meta),
        capture_output=True,
        text=True,
        check=False,
    )
    assert r.returncode == 0, r.stderr + r.stdout
    assert "no runAll source" in (r.stderr + r.stdout)
    assert not (meta / "deploy-binaries" / "runAll").exists()


def test_hook_collects_when_source_present(tmp_path):
    meta = _fake_meta(tmp_path)
    staging = tmp_path / "stage"
    _write(staging / "runAll", "from-hook\n")
    (staging / "runAll").chmod(0o755)
    r = subprocess.run(
        ["bash", str(HOOK)],
        env=_hook_env(meta, COLLECT_STAGING=str(staging)),
        capture_output=True,
        text=True,
        check=False,
    )
    assert r.returncode == 0, r.stderr + r.stdout
    assert (meta / "deploy-binaries" / "runAll").read_text(encoding="utf-8") == "from-hook\n"


def test_hook_default_ram_deploy_is_home_daydaymoney():
    text = HOOK.read_text(encoding="utf-8")
    assert "${RAM_DEPLOY:-${DEPLOY_ROOT:-$HOME/bin/daydaymoney-deploy}}" in text
    assert "/tmp/ram-deploy" not in text


def test_hook_passes_collect_skip_sha_into_collect_env():
    """源码机 ELF 与 releases.yaml Release pin 本就不同：钩子必须导出 COLLECT_SKIP_SHA=1（OPT-20260902-009）。"""
    text = HOOK.read_text(encoding="utf-8")
    assert "COLLECT_SKIP_SHA=1" in text


def test_hook_mismatched_release_pin_still_exit0_no_sha_noise(tmp_path):
    """错 pin 的 releases.yaml + 源码 ELF 时，钩子 exit 0 且不打印 SHA MISMATCH。"""
    meta = _fake_meta(tmp_path)
    staging = tmp_path / "stage"
    _write(staging / "runAll", "from-hook\n")
    rel = meta / "conf.example" / "releases.yaml"
    _write(
        rel,
        "artifacts:\n"
        "  runAll:\n"
        "    package: bin/runAll\n"
        '    sha: "0000000000000000000000000000000000000000000000000000000000000000"\n',
    )
    r = subprocess.run(
        ["bash", str(HOOK)],
        env=_hook_env(meta, COLLECT_STAGING=str(staging)),
        capture_output=True,
        text=True,
        check=False,
    )
    out = r.stdout + r.stderr
    assert r.returncode == 0, out
    assert "SHA MISMATCH" not in out
    assert "skip sha-verify" in out
