#!/usr/bin/env python3
"""
CI guard: every Go service that resolves a MySQL DSN must reference a key
registered in db/registry.yaml.

Finds calls like:
  dbload.ResolveMySQLDSN("key", ...)
  dbload.ResolveMySQLDSN(ctx, "key", ...)
  ResolveMySQLDSN("key", ...)

Reports any key not found in registry.yaml.
Supports known exceptions for testing/legacy code (declared in EXCEPTIONS).

Usage:
  python3 db/scripts/ci/check_database_registry_compliance.py
Output:
  - Exit 0: all referenced keys are in registry
  - Exit 1: one or more unregistered keys found
"""

import os
import re
import sys
from pathlib import Path

import yaml


# Keys known to be referenced in non-service contexts (tests, migrations, scripts).
# These are intentionally excluded from the registry check.
EXCEPTIONS: set[str] = set()


def find_monorepo_root() -> Path:
    """Walk up to find the monorepo root (contains db/registry.yaml)."""
    start = Path(__file__).resolve().parent.parent.parent  # db/scripts/ci -> db -> root
    if (start / "db" / "registry.yaml").is_file():
        return start
    # Fallback: walk up
    d = Path.cwd()
    for _ in range(10):
        if (d / "db" / "registry.yaml").is_file():
            return d
        parent = d.parent
        if parent == d:
            break
        d = parent
    raise FileNotFoundError("db/registry.yaml not found above cwd")


def load_registry_keys(root: Path) -> set[str]:
    """Return the set of database keys from registry.yaml."""
    reg_path = root / "db" / "registry.yaml"
    with open(reg_path) as f:
        doc = yaml.safe_load(f) or {}
    dbs = doc.get("databases") or {}
    return set(dbs.keys())


# Pattern: ResolveMySQLDSN("key" ...) or ResolveMySQLDSN(ctx, "key" ...)
# Captures the first string literal argument after ResolveMySQLDSN(
_RESOLVE_MYSQL_DSN_RE = re.compile(
    r'\b(?:dbload\.)?ResolveMySQLDSN\s*\(\s*(?:[^"\')\s]+\s*,\s*)?["\']([^"\']+)["\']'
)

# Pattern: GetDatabaseName("key" ...)
_GET_DB_NAME_RE = re.compile(
    r'\b(?:dbload\.)?GetDatabaseName\s*\(\s*["\']([^"\']+)["\']'
)


def scan_go_file(path: Path) -> set[str]:
    """Extract all database key references from a Go source file."""
    keys: set[str] = set()
    try:
        text = path.read_text(encoding="utf-8")
    except (OSError, UnicodeDecodeError):
        return keys
    for m in _RESOLVE_MYSQL_DSN_RE.finditer(text):
        keys.add(m.group(1))
    for m in _GET_DB_NAME_RE.finditer(text):
        keys.add(m.group(1))
    return keys


def scan_go_services(root: Path) -> dict[str, set[str]]:
    """Scan all Go service directories for database key references.

    Returns: {service_dir_name: {key1, key2, ...}}
    """
    # Go services are top-level dirs with go.mod
    service_keys: dict[str, set[str]] = {}
    exclude = {
        "go_relayToTrae",
        "trae-agent", "dataMigrate", "commitResult",
        "claude-agent", "taskGateway", "taskContainerGateway",
        "taskSSE", "taskEvents", "taskAgentSupport", "taskAIEndPoint",
    }
    for entry in sorted(root.iterdir()):
        if not entry.is_dir():
            continue
        if entry.name in exclude or entry.name.startswith("."):
            continue
        go_mod = entry / "go.mod"
        if not go_mod.is_file():
            continue
        keys: set[str] = set()
        for go_file in entry.rglob("*.go"):
            # Skip test files and vendor
            if go_file.name.endswith("_test.go"):
                continue
            if "vendor" in go_file.parts or ".git" in go_file.parts:
                continue
            keys |= scan_go_file(go_file)
        if keys:
            service_keys[entry.name] = keys
    return service_keys


def main() -> int:
    root = find_monorepo_root()
    registry_keys = load_registry_keys(root)
    print(f"[registry] {len(registry_keys)} databases registered in db/registry.yaml")

    service_keys = scan_go_services(root)

    errors = 0
    for svc_name, keys in sorted(service_keys.items()):
        unknown = (keys - registry_keys) - EXCEPTIONS
        if unknown:
            for k in sorted(unknown):
                print(f"ERROR: {svc_name} references unregistered database key '{k}' — add it to db/registry.yaml")
                errors += 1

    # Also print summary of found references for audit
    total_refs = sum(len(v) for v in service_keys.values())
    print(f"[compliance] scanned {len(service_keys)} Go services, {total_refs} key references found")

    if errors:
        print(f"\nFAILED: {errors} unregistered database key(s) found.")
        print(f"Add them to db/registry.yaml per .ai/01_project_constraints/38_database_init_registry_mandate.md")
        return 1

    print("PASSED: all referenced database keys are registered in db/registry.yaml")
    return 0


if __name__ == "__main__":
    sys.exit(main())
