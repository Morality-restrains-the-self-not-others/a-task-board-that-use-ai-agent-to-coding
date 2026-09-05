#!/usr/bin/env python3
"""回归：compose down 后再 up，APISIX 冷启动可超过 90s（init_worker ~140s）。

run.sh 不得在 30s 时 exit 1 误报失败；默认等待须覆盖该窗口，且未就绪时
默认只告警（容器已 up），把最终判定交给 runAll health_check。
"""
from __future__ import annotations

import os
import stat
import subprocess
from pathlib import Path

ROOT = Path(__file__).resolve().parents[1]
RUN_SH = ROOT / "run.sh"
RUNALL_YAML = ROOT.parent / "conf" / "runAll.yaml"


def _start_case(text: str) -> str:
    after = text.split("start)", 1)[1]
    return after.split("stop)", 1)[0]


def test_default_ready_timeout_covers_cold_compose():
    text = RUN_SH.read_text(encoding="utf-8")
    assert "TASKGATEWAY_APISIX_READY_TIMEOUT_SEC:-180" in text, (
        "run.sh 默认 APISIX ready 等待须 ≥180s，覆盖 compose recreate 后的冷启动"
    )
    start = _start_case(text)
    assert "seq 1 30" not in start, "禁止把 APISIX ready 轮询写死为 30s"


def test_not_ready_exit_1_is_opt_in():
    start = _start_case(RUN_SH.read_text(encoding="utf-8"))
    assert "TASKGATEWAY_APISIX_READY_FAIL" in start
    # exit 1 必须挂在 FAIL=1 闸门后，不能在 30s 未就绪时无条件失败
    fail_idx = start.find("TASKGATEWAY_APISIX_READY_FAIL")
    exit_idx = start.find("exit 1")
    assert exit_idx > fail_idx, "APISIX 未就绪的 exit 1 必须是 TASKGATEWAY_APISIX_READY_FAIL 可选行为"


def _prepare_body(text: str) -> str:
    start = text.find("prepare_apisix_logs_for_start()")
    assert start >= 0
    return text[start : text.find("\ncmd=", start)]


def test_prepare_logs_chmods_bind_mount_via_docker():
    """P4 上 Docker 会把缺失的 logs bind-mount 建成 root:root 755，uid 636 写不了 error.log。"""
    body = _prepare_body(RUN_SH.read_text(encoding="utf-8"))
    assert "docker run" in body
    assert "chmod 777 /logs" in body


def test_prepare_makes_standalone_config_yaml_writable():
    """APISIX entrypoint: echo "$(sed …)" > config.yaml 以 uid 636 重写 bind-mount。

    seed/rsync 后文件为 0644 属主 1000 → Permission denied → 容器 Exited(1)
    → 宿主机 :18081 connection refused。
    """
    body = _prepare_body(RUN_SH.read_text(encoding="utf-8"))
    assert "apisix/config.yaml" in body, "准备函数须点名 chmod 入口要重写的 config.yaml"
    assert "chmod a+rw" in body


def test_prepare_host_mounts_makes_0644_config_other_writable(tmp_path: Path | None = None):
    base = tmp_path if tmp_path is not None else Path(
        __import__("tempfile").mkdtemp(prefix="gw-prep-")
    )
    dest = base / "gw"
    dest.mkdir()
    (dest / "run.sh").write_bytes(RUN_SH.read_bytes())
    (dest / "run.sh").chmod(0o755)
    (dest / "apisix").mkdir()
    (dest / "logs").mkdir()
    cfg = dest / "apisix" / "config.yaml"
    cfg.write_text("deployment:\n  role: traditional\n", encoding="utf-8")
    cfg.chmod(0o644)
    assert cfg.stat().st_mode & stat.S_IWOTH == 0
    env = dict(os.environ)
    r = subprocess.run(
        ["bash", str(dest / "run.sh"), "prepare-host-mounts"],
        cwd=str(dest),
        env=env,
        capture_output=True,
        text=True,
        check=False,
    )
    assert r.returncode == 0, r.stderr + r.stdout
    assert cfg.stat().st_mode & stat.S_IWOTH, (
        f"config.yaml mode={oct(cfg.stat().st_mode)} still not other-writable; "
        f"stdout={r.stdout!r} stderr={r.stderr!r}"
    )


def test_start_short_circuits_ready_wait_when_container_exited():
    start = _start_case(RUN_SH.read_text(encoding="utf-8"))
    up_idx = start.find("up -d")
    assert up_idx >= 0
    after_up = start[up_idx:]
    assert "taskgateway-apisix-1" in after_up or "TASKGATEWAY_APISIX_CONTAINER" in after_up
    assert "apisix_ready_timeout" in after_up
    assert "not running" in after_up.lower() or "not running after compose" in after_up


def test_runall_task_gateway_health_timeout_covers_cold_apisix():
    if not RUNALL_YAML.is_file():
        return
    text = RUNALL_YAML.read_text(encoding="utf-8")
    marker = "- name: task-gateway"
    i = text.find(marker)
    assert i >= 0, "conf/runAll.yaml 缺少 task-gateway"
    nxt = text.find("\n      - name:", i + len(marker))
    block = text[i : nxt if nxt > i else i + 800]
    assert "timeout: 180" in block, (
        "task-gateway health_check.timeout 须 ≥180，否则 start-all 在 APISIX init_worker 前判失败"
    )


if __name__ == "__main__":
    tests = [v for k, v in sorted(globals().items()) if k.startswith("test_")]
    for t in tests:
        t()
        print(f"PASS {t.__name__}")
    print(f"OK — {len(tests)} tests passed")
