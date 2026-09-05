#!/usr/bin/env python3
"""Generate Prometheus file_sd targets from runAll.yaml health_check URLs."""

from __future__ import annotations

import argparse
import json
import sys
from pathlib import Path

_SCRIPTS = Path(__file__).resolve().parents[2] / "runAll" / "scripts"
if str(_SCRIPTS) not in sys.path:
    sys.path.insert(0, str(_SCRIPTS))
from conf_local import overlay_conf_file  # noqa: E402


def parse_args() -> argparse.Namespace:
    parser = argparse.ArgumentParser(
        description="Generate Prometheus file_sd targets from runAll config."
    )
    parser.add_argument(
        "--runall-config",
        default="../conf/runAll.yaml",
        help="Path to runAll YAML config (default: ../conf/runAll.yaml from AiMonitor/)",
    )
    parser.add_argument(
        "--output",
        default="prometheus/file_sd/runall-health-targets.json",
        help="Output file_sd JSON path (default: prometheus/file_sd/runall-health-targets.json)",
    )
    return parser.parse_args()


def load_yaml(path: Path) -> dict:
    data = overlay_conf_file(path)
    if not isinstance(data, dict):
        raise ValueError(f"Invalid YAML root in {path}")
    return data


def resolve_intent_url(root: Path, service: dict) -> str | None:
    """Build the probe URL for a task-events service from its intent_path SSOT.

    OPT-20260816-053: runAll.yaml no longer hand-writes task-events health ports;
    the port lives in conf/events/domain-events/<event>/config.yaml. This mirrors
    runAll LoadConfig so regenerating Prometheus targets keeps task-events services.
    """
    intent_path = service.get("intent_path")
    health = service.get("health_check") or {}
    if not isinstance(intent_path, str) or not intent_path.strip():
        return None
    health_path = health.get("health_path")
    if not isinstance(health_path, str) or not health_path.strip():
        return None
    parts = [p for p in intent_path.split("/") if p]
    if len(parts) < 2:
        return None
    cfg = root / "conf" / "events" / "domain-events" / parts[-2] / "config.yaml"
    if not cfg.is_file():
        return None
    data = overlay_conf_file(cfg)
    intents = data.get("intents") if isinstance(data, dict) else None
    if not isinstance(intents, dict):
        return None
    blk = intents.get(parts[-1])
    if not isinstance(blk, dict) or not isinstance(blk.get("port"), int):
        return None
    return f"http://${{INFRA_HOST}}:{blk['port']}{health_path}"


def build_targets(runall: dict, domain_events_root: Path | None = None) -> list[dict]:
    groups = runall.get("groups", [])
    if not isinstance(groups, list):
        raise ValueError("runAll config 'groups' must be a list")

    targets: list[dict] = []
    seen: set[tuple[str, str]] = set()

    for group in groups:
        if not isinstance(group, dict):
            continue
        group_name = str(group.get("name", "unknown"))
        services = group.get("services", [])
        if not isinstance(services, list):
            continue

        for service in services:
            if not isinstance(service, dict):
                continue
            service_name = str(service.get("name", "unknown"))
            health = service.get("health_check", {})
            if not isinstance(health, dict):
                continue
            url = health.get("url")
            if not isinstance(url, str) or not url.strip():
                url = (
                    resolve_intent_url(domain_events_root, service)
                    if domain_events_root is not None
                    else None
                )
            if not isinstance(url, str) or not url.strip():
                continue

            key = (service_name, url)
            if key in seen:
                continue
            seen.add(key)

            targets.append(
                {
                    "labels": {
                        "group": group_name,
                        "service": service_name,
                    },
                    "targets": [url.strip()],
                }
            )

    return targets


def main() -> int:
    args = parse_args()
    root = Path(__file__).resolve().parents[1]
    runall_path = (root / args.runall_config).resolve()
    output_path = (root / args.output).resolve()

    runall = load_yaml(runall_path)
    targets = build_targets(runall, domain_events_root=root.parent)
    output_path.parent.mkdir(parents=True, exist_ok=True)
    with output_path.open("w", encoding="utf-8") as f:
        json.dump(targets, f, indent=2, ensure_ascii=False)
        f.write("\n")

    print(f"[ok] generated {len(targets)} targets -> {output_path}")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
