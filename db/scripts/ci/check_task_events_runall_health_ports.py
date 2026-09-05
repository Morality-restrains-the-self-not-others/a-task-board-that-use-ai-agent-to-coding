#!/usr/bin/env python3
"""CI: runAll / Prometheus / intent_registry health ports must match listen SSOT.

SSOT: conf/events/domain-events/<event>/config.yaml → intents.<intent>.port
Probes: conf/runAll.yaml task-events-* services (intent_path → SSOT port)
        AiMonitor/prometheus/file_sd/runall-health-targets.json
        taskEvents/config/intent_registry.go Port + GroupID

OPT-20260816-053: runAll.yaml must NOT hand-write task-events health ports.
Every task-events-* service declares `intent_path: events/domain-events/<event>/<intent-key>`;
runAll LoadConfig resolves the port from the SSOT and builds health_check URLs.
This script cross-checks that intent_path resolves to the intent whose groupId equals
the service name, so copy-pasting a sibling intent (FE-20260816-EVENTS-FANOUT-HEALTH-PORT)
is caught statically and at load time.

New domain-events groupIds must appear in runAll.yaml. A frozen allowlist
covers intents already missing as of 2026-08-16 (OPT-20260816-052); do not grow it.
"""
from __future__ import annotations

import json
import re
import sys
from collections import defaultdict
from pathlib import Path

try:
    import yaml
except ImportError:  # pragma: no cover
    print("ERROR: PyYAML required", file=sys.stderr)
    raise SystemExit(2)

PORT_RE = re.compile(r":(\d+)(?:/|$)")
REGISTRY_RE = re.compile(r'Port:\s*(\d+),\s*GroupID:\s*"([^"]+)"')

# SSOT intents not in runAll.yaml as of 2026-08-16. Shrink only (OPT-20260816-052).
LEGACY_UNMANAGED_GROUP_IDS = frozenset()


def monorepo_root() -> Path:
    here = Path(__file__).resolve()
    for parent in here.parents:
        if (parent / "db" / "registry.yaml").is_file():
            return parent
    raise FileNotFoundError("db/registry.yaml not found")


def _port_from_url(url: object) -> int | None:
    if not isinstance(url, str) or not url.strip():
        return None
    m = PORT_RE.search(url)
    return int(m.group(1)) if m else None


def load_ssot_ports(root: Path) -> dict[str, int]:
    """groupId → listen port from domain-events per-event YAML."""
    base = root / "conf" / "events" / "domain-events"
    out: dict[str, int] = {}
    if not base.is_dir():
        return out
    for cfg_path in sorted(base.glob("*/config.yaml")):
        data = yaml.safe_load(cfg_path.read_text(encoding="utf-8")) or {}
        intents = data.get("intents") if isinstance(data, dict) else None
        if not isinstance(intents, dict):
            continue
        for block in intents.values():
            if not isinstance(block, dict):
                continue
            gid = block.get("groupId")
            port = block.get("port")
            if isinstance(gid, str) and gid.strip() and isinstance(port, int):
                out[gid.strip()] = port
    return out


def iter_task_events_services(root: Path):
    """Yield (name, health_check dict, intent_path str|None) for task-events-* services."""
    path = root / "conf" / "runAll.yaml"
    data = yaml.safe_load(path.read_text(encoding="utf-8")) or {}
    for group in data.get("groups") or []:
        if not isinstance(group, dict):
            continue
        for svc in group.get("services") or []:
            if not isinstance(svc, dict):
                continue
            name = svc.get("name")
            if not isinstance(name, str) or not name.startswith("task-events-"):
                continue
            ip = svc.get("intent_path")
            yield name, svc.get("health_check") or {}, (ip if isinstance(ip, str) else None)


def resolve_intent_port(root: Path, intent_path: str) -> tuple[int, str] | None:
    """Resolve `intent_path` → (port, groupId) from domain-events SSOT.

    Returns None when intent_path is empty or does not resolve to a concrete intent.
    """
    if not intent_path:
        return None
    parts = [p for p in intent_path.split("/") if p]
    if len(parts) < 2:
        return None
    event_dir = parts[-2]
    intent_key = parts[-1]
    cfg = root / "conf" / "events" / "domain-events" / event_dir / "config.yaml"
    if not cfg.is_file():
        return None
    data = yaml.safe_load(cfg.read_text(encoding="utf-8")) or {}
    intents = data.get("intents") if isinstance(data, dict) else None
    if not isinstance(intents, dict):
        return None
    blk = intents.get(intent_key)
    if not isinstance(blk, dict):
        return None
    port = blk.get("port")
    if not isinstance(port, int):
        return None
    gid = blk.get("groupId")
    return port, (gid if isinstance(gid, str) else "")


def load_runall_health_ports(root: Path) -> dict[str, int]:
    """task-events-* service name → SSOT-resolved port (via intent_path)."""
    out: dict[str, int] = {}
    for name, hc, intent_path in iter_task_events_services(root):
        resolved = resolve_intent_port(root, intent_path or "")
        if resolved is not None:
            out[name] = resolved[0]
    return out


def load_prom_health_ports(root: Path) -> dict[str, int]:
    path = root / "AiMonitor" / "prometheus" / "file_sd" / "runall-health-targets.json"
    if not path.is_file():
        return {}
    data = json.loads(path.read_text(encoding="utf-8"))
    out: dict[str, int] = {}
    if not isinstance(data, list):
        return out
    for item in data:
        if not isinstance(item, dict):
            continue
        labels = item.get("labels") or {}
        name = labels.get("service") if isinstance(labels, dict) else None
        if not isinstance(name, str) or not name.startswith("task-events-"):
            continue
        targets = item.get("targets") or []
        if not targets:
            continue
        port = _port_from_url(str(targets[0]))
        if port is not None:
            out[name] = port
    return out


def load_registry_ports(root: Path) -> dict[str, int]:
    path = root / "taskEvents" / "config" / "intent_registry.go"
    if not path.is_file():
        return {}
    text = path.read_text(encoding="utf-8")
    out: dict[str, int] = {}
    for m in REGISTRY_RE.finditer(text):
        out[m.group(2)] = int(m.group(1))
    return out


def mismatches(observed: dict[str, int], ssot: dict[str, int]) -> list[tuple[str, int, int]]:
    bad: list[tuple[str, int, int]] = []
    for name, port in sorted(observed.items()):
        if name not in ssot:
            continue
        if port != ssot[name]:
            bad.append((name, port, ssot[name]))
    return bad


def duplicate_ports(observed: dict[str, int]) -> dict[int, list[str]]:
    by_port: dict[int, list[str]] = defaultdict(list)
    for name, port in observed.items():
        by_port[port].append(name)
    return {p: sorted(names) for p, names in by_port.items() if len(names) > 1}


def unmanaged_group_ids(
    ssot: dict[str, int],
    runall: dict[str, int],
    allow: frozenset[str],
) -> list[str]:
    return sorted(gid for gid in ssot if gid not in runall and gid not in allow)


def report(root: Path, *, allow: frozenset[str] | None = None) -> list[str]:
    frozen = LEGACY_UNMANAGED_GROUP_IDS if allow is None else allow
    ssot = load_ssot_ports(root)
    runall = load_runall_health_ports(root)
    prom = load_prom_health_ports(root)
    registry = load_registry_ports(root)
    errors: list[str] = []
    for name, got, want in mismatches(runall, ssot):
        errors.append(
            f"runAll.yaml {name}: health port {got} != SSOT {want} "
            f"(conf/events/domain-events/*/config.yaml)"
        )
    for name, hc, intent_path in iter_task_events_services(root):
        url_p = _port_from_url(hc.get("url") or "")
        live_p = _port_from_url(hc.get("liveness_url") or "")
        if not intent_path:
            errors.append(
                f"runAll.yaml {name}: missing intent_path "
                f"(health port must come from domain-events SSOT)"
            )
        if url_p is not None or live_p is not None:
            errors.append(
                f"runAll.yaml {name}: hand-written health port "
                f"(url={url_p}, liveness={live_p}) forbidden; derive from intent_path"
            )
        if url_p is not None and live_p is not None and url_p != live_p:
            errors.append(
                f"runAll.yaml {name}: url port {url_p} != liveness_url port {live_p}"
            )
        resolved = resolve_intent_port(root, intent_path or "")
        if resolved is not None:
            port, gid = resolved
            if gid and gid != name:
                errors.append(
                    f"runAll.yaml {name}: intent_path resolves to groupId {gid} (name mismatch, "
                    f"copy-pasted sibling intent?)"
                )
    for name, got, want in mismatches(prom, ssot):
        errors.append(
            f"runall-health-targets.json {name}: target port {got} != SSOT {want}"
        )
    for name, got, want in mismatches(registry, ssot):
        errors.append(
            f"intent_registry.go {name}: Port {got} != SSOT {want}"
        )
    for port, names in sorted(duplicate_ports(runall).items()):
        errors.append(
            f"runAll.yaml duplicate health port {port}: {', '.join(names)}"
        )
    for gid in unmanaged_group_ids(ssot, runall, frozen):
        errors.append(
            f"SSOT intent {gid} has no runAll.yaml service "
            f"(add domain-events-intents entry; do not grow LEGACY_UNMANAGED_GROUP_IDS)"
        )
    return errors


def main(argv: list[str] | None = None) -> int:
    del argv
    root = monorepo_root()
    errors = report(root)
    if errors:
        print("VIOLATION (rule 52_runall_health_port_ssot.md):")
        for line in errors:
            print(f"  {line}")
        return 1
    runall = load_runall_health_ports(root)
    print(f"ok: {len(runall)} task-events health ports match domain-events SSOT")
    return 0


if __name__ == "__main__":
    sys.exit(main())
