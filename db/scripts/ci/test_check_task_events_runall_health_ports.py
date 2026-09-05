#!/usr/bin/env python3
"""Self-test for check_task_events_runall_health_ports (no pytest required).

Reproduces the 2026-08-16 READINESS_TIMEOUT: runAll.yaml probed 18056 while
2_fanout_work_panel_sse listened on 18048 (wechat-identity-conflict's port).

Since OPT-20260816-053 runAll.yaml no longer hand-writes task-events health ports;
services declare intent_path and the port comes from the domain-events SSOT. The
copy-paste failure mode is now a service whose intent_path resolves to a sibling
intent's groupId (name mismatch) / wrong port.
"""
from __future__ import annotations

import importlib.util
import json
import sys
from pathlib import Path

ROOT = Path(__file__).resolve().parents[3]
CHECKER = ROOT / "db" / "scripts" / "ci" / "check_task_events_runall_health_ports.py"

FANOUT = "task-events-task-status-changed-2-fanout-work-panel-sse"
WECHAT = "task-events-wechat-identity-conflict-1-audit-alert"

SSOT_TASK_STATUS = """intents:
  2_fanout_work_panel_sse:
    host: 0.0.0.0
    port: 18048
    groupId: task-events-task-status-changed-2-fanout-work-panel-sse
"""

SSOT_WECHAT = """intents:
  1_audit_alert:
    host: 0.0.0.0
    port: 18056
    groupId: task-events-wechat-identity-conflict-1-audit-alert
"""

# Fanout copy-pasted a sibling intent (wechat-identity-conflict) as its intent_path
# → resolves to 18056 with wechat's groupId (name mismatch + wrong port).
RUNALL_MISMATCH = """groups:
  - name: domain-events-intents
    services:
      - name: task-events-task-status-changed-2-fanout-work-panel-sse
        intent_path: events/domain-events/wechat_identity_conflict/1_audit_alert
        health_check:
          health_path: /api/health/ready
      - name: task-events-wechat-identity-conflict-1-audit-alert
        intent_path: events/domain-events/wechat_identity_conflict/1_audit_alert
        health_check:
          health_path: /api/health/ready
"""

RUNALL_FIXED = """groups:
  - name: domain-events-intents
    services:
      - name: task-events-task-status-changed-2-fanout-work-panel-sse
        intent_path: events/domain-events/task_status_changed/2_fanout_work_panel_sse
        health_check:
          health_path: /api/health/ready
      - name: task-events-wechat-identity-conflict-1-audit-alert
        intent_path: events/domain-events/wechat_identity_conflict/1_audit_alert
        health_check:
          health_path: /api/health/ready
"""


def _load():
    spec = importlib.util.spec_from_file_location(
        "check_task_events_runall_health_ports", CHECKER
    )
    assert spec and spec.loader
    mod = importlib.util.module_from_spec(spec)
    sys.modules[spec.name] = mod
    spec.loader.exec_module(mod)
    return mod


def _write_tree(tmp: Path, runall: str, prom_fanout_port: int) -> None:
    (tmp / "db").mkdir()
    (tmp / "db" / "registry.yaml").write_text("databases: []\n", encoding="utf-8")
    de = tmp / "conf" / "events" / "domain-events" / "task_status_changed"
    de.mkdir(parents=True)
    (de / "config.yaml").write_text(SSOT_TASK_STATUS, encoding="utf-8")
    de = tmp / "conf" / "events" / "domain-events" / "wechat_identity_conflict"
    de.mkdir(parents=True)
    (de / "config.yaml").write_text(SSOT_WECHAT, encoding="utf-8")
    (tmp / "conf" / "runAll.yaml").write_text(runall, encoding="utf-8")
    sd = tmp / "AiMonitor" / "prometheus" / "file_sd"
    sd.mkdir(parents=True)
    payload = [
        {
            "labels": {"service": FANOUT},
            "targets": [f"http://${{INFRA_HOST}}:{prom_fanout_port}/api/health/ready"],
        },
        {
            "labels": {"service": WECHAT},
            "targets": ["http://${INFRA_HOST}:18056/api/health/ready"],
        },
    ]
    (sd / "runall-health-targets.json").write_text(
        json.dumps(payload), encoding="utf-8"
    )


def test_wrong_18056_health_port_is_violation(tmp_path: Path) -> None:
    """Regression: fanout health URL copied from wechat-identity-conflict."""
    mod = _load()
    _write_tree(tmp_path, RUNALL_MISMATCH, 18056)
    errors = mod.report(tmp_path)
    joined = "\n".join(errors)
    assert any("18056" in e and "18048" in e and FANOUT in e for e in errors), joined
    assert any("duplicate health port 18056" in e for e in errors), joined


def test_matching_ports_pass(tmp_path: Path) -> None:
    mod = _load()
    _write_tree(tmp_path, RUNALL_FIXED, 18048)
    errors = mod.report(tmp_path, allow=frozenset())
    assert errors == [], errors


def test_prometheus_mismatch_is_violation(tmp_path: Path) -> None:
    mod = _load()
    _write_tree(tmp_path, RUNALL_FIXED, 18056)
    errors = mod.report(tmp_path, allow=frozenset())
    assert any("runall-health-targets.json" in e and FANOUT in e for e in errors), errors


def test_new_ssot_intent_missing_from_runall_is_violation(tmp_path: Path) -> None:
    """Future intents must be registered in runAll.yaml (the 2026-08-16 gap)."""
    mod = _load()
    _write_tree(tmp_path, RUNALL_FIXED, 18048)
    extra = tmp_path / "conf" / "events" / "domain-events" / "brand_new_event"
    extra.mkdir()
    gid = "task-events-brand-new-event-1-do-stuff"
    (extra / "config.yaml").write_text(
        f"intents:\n  1_do_stuff:\n    port: 18099\n    groupId: {gid}\n",
        encoding="utf-8",
    )
    errors = mod.report(tmp_path, allow=frozenset())
    assert any(gid in e and "no runAll.yaml service" in e for e in errors), errors


def test_legacy_unmanaged_allowlist_is_not_violation(tmp_path: Path) -> None:
    mod = _load()
    _write_tree(tmp_path, RUNALL_FIXED, 18048)
    extra = tmp_path / "conf" / "events" / "domain-events" / "task_created"
    extra.mkdir()
    gid = "task-events-task-created-1-fanout-work-panel-sse"
    (extra / "config.yaml").write_text(
        f"intents:\n  1_fanout_work_panel_sse:\n    port: 18050\n    groupId: {gid}\n",
        encoding="utf-8",
    )
    errors = mod.report(tmp_path, allow=frozenset({gid}))
    assert errors == [], errors


def test_intent_registry_port_mismatch_is_violation(tmp_path: Path) -> None:
    mod = _load()
    _write_tree(tmp_path, RUNALL_FIXED, 18048)
    reg = tmp_path / "taskEvents" / "config"
    reg.mkdir(parents=True)
    (reg / "intent_registry.go").write_text(
        'package config\nvar AllIntents = []IntentDefinition{\n'
        f'\t{{Port: 18056, GroupID: "{FANOUT}"}},\n'
        f'\t{{Port: 18056, GroupID: "{WECHAT}"}},\n'
        "}\n",
        encoding="utf-8",
    )
    errors = mod.report(tmp_path, allow=frozenset())
    assert any("intent_registry.go" in e and FANOUT in e and "18056" in e for e in errors), errors


def test_liveness_url_port_must_match_ready_url(tmp_path: Path) -> None:
    # Hand-written url/liveness_url ports are forbidden after OPT-20260816-053;
    # the check must flag them (and the ready/liveness port split if mismatched).
    mod = _load()
    _write_tree(tmp_path, RUNALL_FIXED, 18048)
    runall = RUNALL_FIXED.replace(
        "health_check:\n          health_path: /api/health/ready",
        "health_check:\n          health_path: /api/health/ready\n"
        '          url: "http://${INFRA_HOST}:18048/api/health/ready"\n'
        '          liveness_url: "http://${INFRA_HOST}:18056/api/health/"',
        1,
    )
    (tmp_path / "conf" / "runAll.yaml").write_text(runall, encoding="utf-8")
    errors = mod.report(tmp_path, allow=frozenset())
    assert any("liveness_url" in e and FANOUT in e for e in errors), errors


if __name__ == "__main__":
    import tempfile

    tests = [v for k, v in sorted(globals().items()) if k.startswith("test_")]
    for t in tests:
        with tempfile.TemporaryDirectory() as d:
            t(Path(d))
        print(f"PASS {t.__name__}")
    print(f"ok ({len(tests)} tests)")
