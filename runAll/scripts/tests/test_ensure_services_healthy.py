"""ensure_services_healthy watchdog — bulk / lifecycle 门禁单测。"""

from __future__ import annotations

import importlib.util
from pathlib import Path

import pytest

SCRIPT = Path(__file__).resolve().parents[1] / "ensure_services_healthy.py"


def _load_mod():
    spec = importlib.util.spec_from_file_location("ensure_services_healthy", SCRIPT)
    mod = importlib.util.module_from_spec(spec)
    assert spec.loader is not None
    spec.loader.exec_module(mod)
    return mod


@pytest.fixture(scope="module")
def mod():
    return _load_mod()


def test_orchestration_blocks_on_active_bulk(mod, tmp_path):
    status = {"active_bulk_progress": {"kind": "precise-restart"}, "services": []}
    reason = mod.orchestration_blocks_restart(tmp_path, status, "task-auth")
    assert "precise-restart" in reason


def test_orchestration_blocks_on_building_status(mod, tmp_path):
    status = {
        "active_bulk_progress": None,
        "services": [{"name": "task-auth", "status": "building"}],
    }
    reason = mod.orchestration_blocks_restart(tmp_path, status, "task-auth")
    assert "building" in reason


def test_orchestration_allows_healthy_idle(mod, tmp_path):
    status = {
        "active_bulk_progress": None,
        "services": [{"name": "task-auth", "status": "healthy"}],
    }
    assert mod.orchestration_blocks_restart(tmp_path, status, "task-auth") == ""


def test_orchestration_fail_open_without_status(mod, tmp_path):
    assert mod.orchestration_blocks_restart(tmp_path, None, "task-auth") == ""


# OPT-20260810-042：本地 bulk 锁优先于 API 判定（API 不可达时仍能感知）。
def test_orchestration_blocks_on_lock_file_when_status_none(mod, tmp_path):
    import json
    import time

    (tmp_path / ".runall").mkdir()
    (tmp_path / ".runall" / "bulk_op.lock").write_text(
        json.dumps({"kind": "stop-all", "ts": int(time.time())}), encoding="utf-8"
    )
    reason = mod.orchestration_blocks_restart(tmp_path, None, "task-auth")
    assert "stop-all" in reason
    assert "lock" in reason


# 过期锁（>2h 残留）应忽略并回退 API；无 status 时 fail-open。
def test_orchestration_ignores_stale_lock(mod, tmp_path, monkeypatch):
    import json
    import time

    (tmp_path / ".runall").mkdir()
    stale = int(time.time()) - 3 * 3600
    (tmp_path / ".runall" / "bulk_op.lock").write_text(
        json.dumps({"kind": "restart-all", "ts": stale}), encoding="utf-8"
    )
    alerts = []
    monkeypatch.setattr(mod, "alert", lambda _root, msg: alerts.append(msg))
    reason = mod.orchestration_blocks_restart(tmp_path, None, "task-auth")
    assert reason == ""
    assert alerts and "stale" in alerts[0]


# 损坏锁文件视为无锁，回退 API。
def test_orchestration_lock_corrupt_falls_back(mod, tmp_path):
    (tmp_path / ".runall").mkdir(parents=True)
    (tmp_path / ".runall" / "bulk_op.lock").write_text("not-json{{{", encoding="utf-8")
    status = {"active_bulk_progress": None, "services": []}
    assert mod.orchestration_blocks_restart(tmp_path, status, "task-auth") == ""


def test_check_service_skips_restart_during_bulk(mod, tmp_path, monkeypatch):
    root = tmp_path
    (root / "bin").mkdir()
    (root / "bin" / "taskAuth").write_text("#!/bin/sh\n", encoding="utf-8")
    (root / "bin" / "taskAuth").chmod(0o755)

    started = {"n": 0}

    def fake_start(_root, _svc):
        started["n"] += 1
        return True

    monkeypatch.setattr(mod, "is_healthy", lambda _url: False)
    monkeypatch.setattr(mod, "port_listening", lambda _url: False)
    monkeypatch.setattr(mod, "binary_present", lambda _root, _svc: True)
    monkeypatch.setattr(mod, "start_service", fake_start)
    monkeypatch.setattr(mod, "log", lambda *_a, **_k: None)

    svc = {
        "name": "task-auth",
        "url": "http://127.0.0.1:8003/api/health/",
        "start_command": "./bin/taskAuth",
        "working_dir": ".",
        "timeout": 5,
    }
    status = {"active_bulk_progress": {"kind": "precise-restart"}, "services": []}
    line = mod.check_service(root, svc, verbose=False, runall_status=status)
    assert line.startswith("SKIP")
    assert started["n"] == 0


def test_check_service_restarts_when_idle(mod, tmp_path, monkeypatch):
    root = tmp_path
    started = {"n": 0}

    def fake_start(_root, _svc):
        started["n"] += 1
        return True

    monkeypatch.setattr(mod, "is_healthy", lambda _url: False)
    monkeypatch.setattr(mod, "port_listening", lambda _url: False)
    monkeypatch.setattr(mod, "binary_present", lambda _root, _svc: True)
    monkeypatch.setattr(mod, "start_service", fake_start)
    monkeypatch.setattr(mod, "wait_healthy", lambda _url, _timeout: True)
    monkeypatch.setattr(mod, "log", lambda *_a, **_k: None)

    svc = {
        "name": "task-auth",
        "url": "http://127.0.0.1:8003/api/health/",
        "start_command": "./bin/taskAuth",
        "working_dir": ".",
        "timeout": 5,
    }
    status = {
        "active_bulk_progress": None,
        "services": [{"name": "task-auth", "status": "failed"}],
    }
    line = mod.check_service(root, svc, verbose=False, runall_status=status)
    assert "RESTARTED" in line
    assert started["n"] == 1


# ── OPT-20260813-006: runAll UI 进程守护 ─────────────────────────────

def test_runall_recovery_allowed_cooldown(mod, tmp_path):
    import time

    (tmp_path / ".runall").mkdir()
    now = int(time.time())
    # 无戳 → 允许
    assert mod.runall_recovery_allowed(tmp_path, now=now)
    # 刚标记 → 冷却中
    mod.mark_runall_recovery(tmp_path, now=now)
    assert not mod.runall_recovery_allowed(tmp_path, now=now + mod.RUNALL_RECOVER_COOLDOWN_SEC - 1)
    # 冷却期满 → 允许
    assert mod.runall_recovery_allowed(tmp_path, now=now + mod.RUNALL_RECOVER_COOLDOWN_SEC)


def test_runall_ui_up_probes_url(mod, monkeypatch):
    import urllib.error

    class FakeResp:
        status = 200

        def __enter__(self):
            return self

        def __exit__(self, *exc):
            return False

    calls = {}

    def fake_urlopen(url, timeout=2):
        calls["url"] = url
        return FakeResp()

    monkeypatch.setattr(mod.urllib.request, "urlopen", fake_urlopen)
    assert mod.runall_ui_up("http://127.0.0.1:9999/")
    assert calls["url"] == "http://127.0.0.1:9999/"

    def fake_error(url, timeout=2):
        raise urllib.error.URLError("down")

    monkeypatch.setattr(mod.urllib.request, "urlopen", fake_error)
    assert not mod.runall_ui_up()


def test_spawn_runall_missing_binary(mod, tmp_path):
    (tmp_path / "runAll" / "bin").mkdir(parents=True)
    # bin/runAll 不存在 → 返回 False，不炸
    assert mod.spawn_runall(tmp_path) is False


def test_spawn_runall_spawns_binary(mod, tmp_path, monkeypatch):
    import subprocess as _sub

    real_popen = _sub.Popen

    (tmp_path / "runAll" / "bin").mkdir(parents=True)
    (tmp_path / "runAll" / "bin" / "runAll").write_text("#!/bin/sh\n", encoding="utf-8")
    spawned = {}

    def fake_popen(args, cwd=None, stdout=None, stderr=None, stdin=None,
                   start_new_session=False, env=None):
        spawned["args"] = args
        spawned["cwd"] = cwd
        return real_popen(["true"], stdout=_sub.DEVNULL, stderr=_sub.DEVNULL)

    monkeypatch.setattr(mod.subprocess, "Popen", fake_popen)
    monkeypatch.setattr(mod, "log_dir", lambda _root: tmp_path / "logs")
    assert mod.spawn_runall(tmp_path) is True
    assert spawned["args"][0] == "setsid"
    assert spawned["cwd"] == str(tmp_path / "runAll")
    assert "--config" in spawned["args"]
    assert spawned["args"][-1] == "../conf/runAll.yaml"


def test_trigger_start_all_polls_and_posts(mod, tmp_path, monkeypatch):
    import json as _json

    posted = {}

    class FakeResp:
        def __enter__(self):
            return self

        def __exit__(self, *exc):
            return False

        def read(self):
            return _json.dumps({"run_id": "start-all-watchdog-x"}).encode("utf-8")

    def fake_ui_up(url=None):
        return True

    def fake_urlopen(req, timeout=5):
        import json as _j
        posted["url"] = req.full_url
        posted["body"] = _j.loads(req.data.decode("utf-8"))
        return FakeResp()

    monkeypatch.setattr(mod, "runall_ui_up", fake_ui_up)
    monkeypatch.setattr(mod.urllib.request, "urlopen", fake_urlopen)
    assert mod.trigger_start_all(tmp_path) is True
    assert posted["url"].endswith("/api/start-all")
    assert posted["body"]["session_id"].startswith("watchdog-")


def test_recover_runall_ok_when_up(mod, tmp_path, monkeypatch):
    monkeypatch.setattr(mod, "runall_ui_up", lambda url=None: True)
    assert mod.recover_runall(tmp_path, restart=True, verbose=False) == ""


def test_recover_runall_disabled_restart(mod, tmp_path, monkeypatch):
    alerts = []
    monkeypatch.setattr(mod, "runall_ui_up", lambda url=None: False)
    monkeypatch.setattr(mod, "alert", lambda _root, msg: alerts.append(msg))
    line = mod.recover_runall(tmp_path, restart=False, verbose=False)
    assert "DOWN" in line
    assert any("disabled" in a for a in alerts)


def test_recover_runall_cooldown_skips(mod, tmp_path, monkeypatch):
    import time

    alerts = []
    (tmp_path / ".runall").mkdir()
    now = int(time.time())
    monkeypatch.setattr(mod, "runall_ui_up", lambda url=None: False)
    monkeypatch.setattr(mod, "runall_recovery_allowed", lambda _root, now=None: False)
    monkeypatch.setattr(mod, "alert", lambda _root, msg: alerts.append(msg))
    line = mod.recover_runall(tmp_path, restart=True, verbose=False)
    assert "COOLDOWN" in line


def test_recover_runall_full_flow(mod, tmp_path, monkeypatch):
    """runAll 失联 → 重启成功 + start-all accepted → RESTARTED。"""
    alerts = []
    calls = {"spawn": 0, "mark": 0, "start_all": 0}
    monkeypatch.setattr(mod, "runall_ui_up", lambda url=None: False)
    monkeypatch.setattr(mod, "runall_recovery_allowed", lambda _root, now=None: True)
    monkeypatch.setattr(mod, "spawn_runall", lambda _root: (calls.update(spawn=calls["spawn"] + 1) or True))
    monkeypatch.setattr(mod, "mark_runall_recovery", lambda _root, now=None: calls.update(mark=calls["mark"] + 1))
    monkeypatch.setattr(mod, "trigger_start_all", lambda _root: (calls.update(start_all=calls["start_all"] + 1) or True))
    monkeypatch.setattr(mod, "alert", lambda _root, msg: alerts.append(msg))
    monkeypatch.setattr(mod, "log", lambda *_a, **_k: None)
    line = mod.recover_runall(tmp_path, restart=True, verbose=False)
    assert "RESTARTED" in line
    assert calls["spawn"] == 1 and calls["mark"] == 1 and calls["start_all"] == 1
    assert any("restarting + start-all" in a for a in alerts)


def test_recover_runall_missing_binary(mod, tmp_path, monkeypatch):
    alerts = []
    monkeypatch.setattr(mod, "runall_ui_up", lambda url=None: False)
    monkeypatch.setattr(mod, "runall_recovery_allowed", lambda _root, now=None: True)
    monkeypatch.setattr(mod, "spawn_runall", lambda _root: False)
    monkeypatch.setattr(mod, "alert", lambda _root, msg: alerts.append(msg))
    line = mod.recover_runall(tmp_path, restart=True, verbose=False)
    assert "BUILD" in line


def test_recover_runall_default_does_not_spawn(mod, tmp_path, monkeypatch):
    spawned = []
    alerts = []
    monkeypatch.setattr(mod, "runall_ui_up", lambda url=None: False)
    monkeypatch.setattr(mod, "spawn_runall", lambda _root: spawned.append(1) or True)
    monkeypatch.setattr(mod, "alert", lambda _root, msg: alerts.append(msg))
    line = mod.recover_runall(tmp_path, verbose=False)
    assert spawned == []
    assert "DOWN" in line
    assert any("disabled" in a for a in alerts)


def test_recover_runall_flag_file_inhibits_opt_in(mod, tmp_path, monkeypatch):
    (tmp_path / ".runall").mkdir()
    (tmp_path / ".runall" / "no_restart_runall").write_text("notify\n", encoding="utf-8")
    spawned = []
    alerts = []
    monkeypatch.setattr(mod, "runall_ui_up", lambda url=None: False)
    monkeypatch.setattr(mod, "spawn_runall", lambda _root: spawned.append(1) or True)
    monkeypatch.setattr(mod, "alert", lambda _root, msg: alerts.append(msg))
    line = mod.recover_runall(tmp_path, restart=True, verbose=False)
    assert spawned == []
    assert "DOWN" in line
    assert any("no_restart_runall" in a or "disabled" in a for a in alerts)


def test_recover_runall_alert_includes_last_exit_fingerprint(mod, tmp_path, monkeypatch):
    # OPT-20260905-005
    (tmp_path / ".runall").mkdir()
    (tmp_path / ".runall" / "last_orchestrator_exit.json").write_text(
        '{"exited_at":"2026-09-05T08:00:00+08:00","pid":4242,'
        '"source":"signal","detail":"terminated"}\n',
        encoding="utf-8",
    )
    alerts = []
    monkeypatch.setattr(mod, "runall_ui_up", lambda url=None: False)
    monkeypatch.setattr(mod, "alert", lambda _root, msg: alerts.append(msg))
    line = mod.recover_runall(tmp_path, verbose=False)
    assert "DOWN" in line
    assert alerts, "expected alert"
    assert any("last_exit(" in a and "source=signal" in a and "pid=4242" in a for a in alerts), alerts


def test_format_last_orchestrator_exit_summary_missing(mod, tmp_path):
    assert mod.format_last_orchestrator_exit_summary(tmp_path) == ""


def test_format_last_orchestrator_exit_summary_unreadable(mod, tmp_path):
    (tmp_path / ".runall").mkdir()
    (tmp_path / ".runall" / "last_orchestrator_exit.json").write_text("{not-json", encoding="utf-8")
    assert mod.format_last_orchestrator_exit_summary(tmp_path) == "last_exit=unreadable"


def test_watchdog_cron_line_disables_runall_restart(mod, tmp_path):
    line = mod.watchdog_cron_line(tmp_path)
    assert "ensure_services_healthy.py --no-restart-runall" in line
    assert str(tmp_path) in line


# opt-in --restart-runall 仍要求冷却 < cron */5，避免 opt-in 后永远撞冷却。
def test_runall_recover_cooldown_below_cron_interval(mod):
    assert mod.RUNALL_RECOVER_COOLDOWN_SEC < 300, (
        "RUNALL_RECOVER_COOLDOWN_SEC 必须 < 300s（cron */5），否则 5 分钟周期的 watchdog "
        "永远撞上冷却，runAll 失联后无法自动拉起"
    )
