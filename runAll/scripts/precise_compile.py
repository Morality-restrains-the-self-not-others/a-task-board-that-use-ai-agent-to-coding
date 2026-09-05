#!/usr/bin/env python3
"""Resolve runAll.yaml services for source-tree precise compile (ADR-0052)."""

from __future__ import annotations

import argparse
import json
import sys
from pathlib import Path

try:
    import yaml
except ImportError:  # pragma: no cover
    print("precise_compile: PyYAML required", file=sys.stderr)
    raise SystemExit(2)


def working_dir_alias(wd: str) -> str:
    wd = (wd or "").strip()
    if not wd or wd == ".":
        return ""
    return wd.split("/", 1)[0]


def load_services(yaml_text: str) -> list[dict[str, str]]:
    data = yaml.safe_load(yaml_text) or {}
    out: list[dict[str, str]] = []
    for group in data.get("groups") or []:
        if not isinstance(group, dict):
            continue
        for svc in group.get("services") or []:
            if not isinstance(svc, dict):
                continue
            name = str(svc.get("name") or "").strip()
            if not name:
                continue
            wd = str(svc.get("working_dir") or "").strip()
            out.append(
                {
                    "name": name,
                    "working_dir": wd,
                    "alias": working_dir_alias(wd),
                    "build_command": str(svc.get("build_command") or "").strip(),
                }
            )
    return out


def read_registry(path: Path) -> list[str]:
    if not path.is_file():
        return []
    names: list[str] = []
    seen: set[str] = set()
    for raw in path.read_text(encoding="utf-8").splitlines():
        line = raw.strip()
        if not line or line.startswith("#"):
            continue
        if line not in seen:
            seen.add(line)
            names.append(line)
    return names


def _known_tokens(services: list[dict[str, str]]) -> set[str]:
    known: set[str] = set()
    for svc in services:
        known.add(svc["name"])
        alias = svc.get("alias") or ""
        if alias:
            known.add(alias)
    return known


def select_services(
    services: list[dict[str, str]],
    requested: list[str] | None,
    *,
    all_buildable: bool = False,
) -> list[dict[str, str]]:
    buildable = [s for s in services if s.get("build_command")]
    if all_buildable:
        return buildable
    if not requested:
        return []
    wanted = {n.strip() for n in requested if n and str(n).strip()}
    out: list[dict[str, str]] = []
    seen: set[str] = set()
    for svc in buildable:
        if svc["name"] in wanted or (svc.get("alias") and svc["alias"] in wanted):
            if svc["name"] not in seen:
                seen.add(svc["name"])
                out.append(svc)
    return out


def unmatched_names(services: list[dict[str, str]], requested: list[str]) -> list[str]:
    known = _known_tokens(services)
    return [n for n in requested if n and n not in known]


def main(argv: list[str] | None = None) -> int:
    parser = argparse.ArgumentParser(description="List source-tree compile targets")
    parser.add_argument("--conf", required=True, help="path to conf/runAll.yaml")
    parser.add_argument("--registry", default="", help="precise_restart_services.txt")
    parser.add_argument("--names", nargs="*", default=[], help="runAll name or working_dir alias")
    parser.add_argument("--all", action="store_true", dest="all_buildable")
    parser.add_argument("--json", action="store_true")
    args = parser.parse_args(argv)

    conf_path = Path(args.conf)
    if not conf_path.is_file():
        print(f"precise_compile: missing {conf_path}", file=sys.stderr)
        return 2
    services = load_services(conf_path.read_text(encoding="utf-8"))

    requested: list[str] = list(args.names)
    if not args.all_buildable and not requested:
        if args.registry:
            requested = read_registry(Path(args.registry))
        if not requested:
            print(
                "precise_compile: no services (pass names, --all, or register via "
                "scripts/register-precise-restart.sh)",
                file=sys.stderr,
            )
            return 2

    if requested:
        unknown = unmatched_names(services, requested)
        if unknown:
            print(
                "precise_compile: unknown service(s): " + ", ".join(unknown),
                file=sys.stderr,
            )
            return 2

    rows = select_services(
        services,
        None if args.all_buildable else requested,
        all_buildable=args.all_buildable,
    )
    if args.json:
        json.dump(rows, sys.stdout)
        sys.stdout.write("\n")
    else:
        for row in rows:
            print("\t".join((row["name"], row["working_dir"], row["build_command"])))
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
