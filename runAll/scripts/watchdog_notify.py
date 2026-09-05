"""Watchdog notify helpers: crontab line and .runall/no_restart_runall flag."""

from __future__ import annotations

import json
import subprocess
import sys
import time
from pathlib import Path

NO_RESTART_RUNALL_FLAG = ".runall/no_restart_runall"
LAST_ORCHESTRATOR_EXIT_FILE = ".runall/last_orchestrator_exit.json"


def watchdog_cron_line(root: Path) -> str:
    return (f"*/5 * * * * cd {root} && python3 runAll/scripts/ensure_services_healthy.py "
            f"--no-restart-runall >> logs/service-watchdog-cron.log 2>&1")


def runall_restart_inhibited(root: Path) -> bool:
    return (root / NO_RESTART_RUNALL_FLAG).is_file()


def write_no_restart_runall_flag(root: Path, reason: str = "") -> Path:
    path = root / NO_RESTART_RUNALL_FLAG
    path.parent.mkdir(parents=True, exist_ok=True)
    body = reason.strip() or "no auto-restart runAll when :9999 is down"
    path.write_text(f"{body}\nwritten {time.strftime('%F %T')}\n", encoding="utf-8")
    return path


def format_last_orchestrator_exit_summary(root: Path) -> str:
    """Compact summary of .runall/last_orchestrator_exit.json for :9999-down alerts.

    OPT-20260905-005: console logs may be truncated; this fingerprint survives.
    Returns empty string when missing (caller omits the clause).
    """
    path = root / LAST_ORCHESTRATOR_EXIT_FILE
    try:
        raw = path.read_text(encoding="utf-8")
    except OSError:
        return ""
    try:
        data = json.loads(raw)
    except (json.JSONDecodeError, TypeError, ValueError):
        return "last_exit=unreadable"
    if not isinstance(data, dict):
        return "last_exit=unreadable"
    source = str(data.get("source") or "unknown").strip() or "unknown"
    detail = str(data.get("detail") or "").strip()
    exited_at = str(data.get("exited_at") or "").strip()
    pid = data.get("pid")
    parts = [f"source={source}"]
    if detail:
        parts.append(f"detail={detail}")
    if exited_at:
        parts.append(f"at={exited_at}")
    if isinstance(pid, int):
        parts.append(f"pid={pid}")
    return "last_exit(" + " ".join(parts) + ")"


def install_cron(root: Path) -> bool:
    """幂等安装/更新 crontab：watchdog 行必须带 --no-restart-runall。"""
    line = watchdog_cron_line(root)
    crontab = subprocess.run(["crontab", "-l"], capture_output=True, text=True, check=False)
    existing = crontab.stdout if crontab.returncode == 0 else ""
    found = False
    out: list[str] = []
    for existing_line in existing.splitlines():
        if "ensure_services_healthy.py" in existing_line:
            out.append(line)
            found = True
        else:
            out.append(existing_line)
    if found:
        if line in existing and existing.count("ensure_services_healthy.py") == 1:
            print("crontab entry already present — no change")
            return True
        new = "\n".join(out) + "\n"
    else:
        marker = "# ram-work-maintenance-cron"
        new = existing.rstrip() + (f"\n{marker}\n{line}\n" if marker not in existing else f"\n{line}\n")
    proc = subprocess.run(["crontab", "-"], input=new, text=True, capture_output=True, check=False)
    if proc.returncode != 0:
        print(f"crontab install failed: {proc.stderr}", file=sys.stderr)
        return False
    print(f"crontab {'updated' if found else 'installed'}: {line}")
    return True
