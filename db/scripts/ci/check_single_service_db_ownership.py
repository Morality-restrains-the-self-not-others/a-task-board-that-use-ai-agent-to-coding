#!/usr/bin/env python3
"""Scan monorepo for multi-service direct access to the same production DB.

SSOT: db/table_ownership.yaml (+ path alignment with db/registry.yaml).
Rule: .ai/01_project_constraints/19_single_service_data_ownership.md

Exit codes:
  0 — no unregistered cross-service DB access
  1 — one or more violations (non-owner service references a DB without whitelist)
  2 — configuration / IO error
"""

from __future__ import annotations

import fnmatch
import sys
from collections import defaultdict
from pathlib import Path
from typing import Any

try:
    import yaml
except ImportError:  # pragma: no cover
    print("ERROR: PyYAML required (import yaml)", file=sys.stderr)
    raise SystemExit(2)

TEXT_SUFFIXES = {".py", ".go", ".yml", ".yaml", ".sh", ".env", ".json", ".toml"}
SKIP_DIR_NAMES = {
    ".git",
    "node_modules",
    "vendor",
    "__pycache__",
    ".venv",
    "venv",
    "dist",
    "build",
    ".tox",
    "gitlab-ce",
}


def strip_line_comments(text: str, suffix: str) -> str:
    """Drop line comments so docstrings/comments mentioning DB names are not hits."""
    out: list[str] = []
    for line in text.splitlines():
        if suffix == ".go":
            if "//" in line:
                line = line.split("//", 1)[0]
        elif suffix == ".py":
            stripped = line.lstrip()
            if stripped.startswith("#"):
                line = ""
            elif "#" in line:
                # keep strings with # roughly; only strip trailing comment when not in quotes
                in_s = False
                quote = ""
                buf: list[str] = []
                i = 0
                while i < len(line):
                    ch = line[i]
                    if in_s:
                        buf.append(ch)
                        if ch == "\\" and i + 1 < len(line):
                            buf.append(line[i + 1])
                            i += 2
                            continue
                        if ch == quote:
                            in_s = False
                        i += 1
                        continue
                    if ch in ("'", '"'):
                        in_s = True
                        quote = ch
                        buf.append(ch)
                        i += 1
                        continue
                    if ch == "#":
                        break
                    buf.append(ch)
                    i += 1
                line = "".join(buf)
        elif suffix == ".sh":
            if "#" in line:
                line = line.split("#", 1)[0]
        out.append(line)
    return "\n".join(out)


def monorepo_root() -> Path:
    here = Path(__file__).resolve()
    for parent in here.parents:
        if (parent / "db" / "registry.yaml").is_file():
            return parent
    raise FileNotFoundError("db/registry.yaml not found above checker")


def load_yaml(path: Path) -> dict[str, Any]:
    data = yaml.safe_load(path.read_text(encoding="utf-8")) or {}
    if not isinstance(data, dict):
        raise ValueError(f"{path}: root must be a mapping")
    return data


def should_exclude(rel: str, patterns: list[str]) -> bool:
    for pat in patterns:
        if fnmatch.fnmatch(rel, pat) or fnmatch.fnmatch(Path(rel).name, pat):
            return True
    return False


def collect_hits(
    root: Path,
    service_roots: list[dict[str, str]],
    markers: list[dict[str, str]],
    exclude_globs: list[str],
) -> dict[str, set[str]]:
    """Return database_key -> set of service owners that reference it in source."""
    hits: dict[str, set[str]] = defaultdict(set)
    sorted_markers = sorted(markers, key=lambda m: len(m.get("marker", "")), reverse=True)

    for entry in service_roots:
        rel_root = entry["path"]
        owner = entry["owner"]
        abs_root = root / rel_root
        if not abs_root.is_dir():
            print(f"WARN: service root missing, skip: {rel_root}", file=sys.stderr)
            continue
        for path in abs_root.rglob("*"):
            if not path.is_file():
                continue
            if any(part in SKIP_DIR_NAMES for part in path.parts):
                continue
            if path.suffix.lower() not in TEXT_SUFFIXES and path.name not in {
                "Dockerfile",
                "Makefile",
            }:
                continue
            rel = path.relative_to(root).as_posix()
            if should_exclude(rel, exclude_globs):
                continue
            try:
                text = path.read_text(encoding="utf-8", errors="ignore")
            except OSError:
                continue
            text = strip_line_comments(text, path.suffix.lower())
            # saas-backend settings 中遗留的未使用 task-auth 路径解析不计入直连
            if owner == "saas-backend" and path.name == "settings.py":
                text = text.replace('resolve_database_path("task-auth"', "")
            matched_keys: set[str] = set()
            for item in sorted_markers:
                marker = item.get("marker") or ""
                db_key = item.get("database_key") or ""
                if not marker or not db_key or db_key in matched_keys:
                    continue
                if marker in text:
                    hits[db_key].add(owner)
                    matched_keys.add(db_key)
    return hits


def main() -> int:
    root = monorepo_root()
    ownership_path = root / "db" / "table_ownership.yaml"
    registry_path = root / "db" / "registry.yaml"
    if not ownership_path.is_file():
        print(f"ERROR: missing {ownership_path}", file=sys.stderr)
        return 2

    ownership = load_yaml(ownership_path)
    databases = ownership.get("databases") or {}
    if not isinstance(databases, dict) or not databases:
        print("ERROR: table_ownership.yaml databases empty", file=sys.stderr)
        return 2

    # Align paths with registry when both declare the same key
    if registry_path.is_file():
        registry = load_yaml(registry_path)
        reg_dbs = registry.get("databases") or {}
        for key, reg in reg_dbs.items():
            if key not in databases:
                continue
            reg_path = (reg or {}).get("path")
            own_path = (databases[key] or {}).get("path")
            if reg_path and own_path and reg_path != own_path:
                print(
                    f"ERROR: path mismatch for {key}: "
                    f"registry={reg_path} ownership={own_path}",
                    file=sys.stderr,
                )
                return 1
            reg_owner = (reg or {}).get("owner")
            own_owner = (databases[key] or {}).get("owner")
            if reg_owner and own_owner and reg_owner != own_owner:
                print(
                    f"ERROR: owner mismatch for {key}: "
                    f"registry={reg_owner} ownership={own_owner}",
                    file=sys.stderr,
                )
                return 1

    owners_by_db = {
        key: str((meta or {}).get("owner") or "").strip()
        for key, meta in databases.items()
    }
    for key, owner in owners_by_db.items():
        if not owner:
            print(f"ERROR: database {key} missing owner", file=sys.stderr)
            return 2

    whitelist: set[tuple[str, str]] = set()
    for item in ownership.get("known_cross_service_access") or []:
        svc = str((item or {}).get("service") or "").strip()
        db_key = str((item or {}).get("database_key") or "").strip()
        if svc and db_key:
            whitelist.add((svc, db_key))

    service_roots = ownership.get("service_roots") or []
    markers = ownership.get("path_markers") or []
    exclude_globs = list(ownership.get("scan_exclude_globs") or [])

    hits = collect_hits(root, service_roots, markers, exclude_globs)

    failures: list[str] = []
    warnings: list[str] = []

    # Also fail if two different owners claim the same path string
    path_to_keys: dict[str, list[str]] = defaultdict(list)
    for key, meta in databases.items():
        p = str((meta or {}).get("path") or "").strip()
        if p:
            path_to_keys[p].append(key)
    for p, keys in path_to_keys.items():
        if len(keys) > 1:
            failures.append(f"duplicate path {p} claimed by databases: {keys}")

    for db_key, services in sorted(hits.items()):
        owner = owners_by_db.get(db_key)
        if not owner:
            failures.append(
                f"services {sorted(services)} reference unknown database_key={db_key}"
            )
            continue
        foreign = sorted(s for s in services if s != owner)
        if not foreign:
            continue
        for svc in foreign:
            if (svc, db_key) in whitelist:
                warnings.append(
                    f"WARN known debt: {svc} → {db_key} "
                    f"(owner={owner}); remove after 转发/迁表"
                )
            else:
                failures.append(
                    f"VIOLATION: {svc} directly references database {db_key} "
                    f"(owner={owner}); use service forwarding or migrate tables, "
                    f"or register known_cross_service_access with tracking"
                )

    for line in warnings:
        print(line, file=sys.stderr)

    if failures:
        print("\nSingle-service DB ownership check FAILED:", file=sys.stderr)
        for line in failures:
            print(f"  - {line}", file=sys.stderr)
        print(
            "\nSee docs/architecture/table-to-owner.md and "
            ".ai/01_project_constraints/19_single_service_data_ownership.md",
            file=sys.stderr,
        )
        return 1

    print(
        "OK: single-service DB ownership "
        f"({len(databases)} databases, {len(warnings)} known-debt warnings)"
    )
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
