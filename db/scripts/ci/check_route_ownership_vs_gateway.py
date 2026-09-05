#!/usr/bin/env python3
"""
CI 门禁：校验 db/api_route_ownership.yaml 中 status: go 的公网前缀已登记进
taskGateway/routes/routes.yaml，防止「Go 已实现、网关白名单漏登 → django-default 404」。

用法：
  python3 db/scripts/ci/check_route_ownership_vs_gateway.py [--repo-root /path/to/repo]

环境变量：
  REPO_ROOT — 覆盖仓库根目录路径。
"""

from __future__ import annotations

import argparse
import os
import sys
from pathlib import Path
from fnmatch import fnmatch


def find_repo_root() -> Path:
    env = os.environ.get("REPO_ROOT", "")
    if env:
        return Path(env)
    return Path(__file__).resolve().parents[3]


def load_yaml_lite(path: Path) -> dict:
    """Load a small YAML using only stdlib (no PyYAML dependency)."""
    import re

    text = path.read_text(encoding="utf-8")

    # Parse top-level scalars, lists, simple mappings
    result: dict = {}
    current_key: str | None = None
    current_list: list = []
    in_list = False
    indent = 0

    for line in text.splitlines():
        stripped = line.strip()
        if not stripped or stripped.startswith("#"):
            continue

        # Top-level key: value
        m = re.match(r'^(\S[^:]*):\s*(.*)$', line)
        if m:
            k = m.group(1).strip()
            v = m.group(2).strip()
            if v:
                # Skip non-data keys
                if k in ("version", "policy", "django_url_scan_roots", "scan_exclude_globs",
                         "approved_python_exceptions", "django_baseline_routes", "upstreams", "routes"):
                    result[k] = True  # placeholder
                continue
            else:
                # Start of a list/mapping
                current_key = k
                current_list = []
                in_list = True
                continue

        if in_list and stripped.startswith("- "):
            item = stripped[2:].strip()
            current_list.append(item)

    return result


def parse_ownership_yaml(path: Path) -> list[dict]:
    """Extract route_prefixes entries from api_route_ownership.yaml."""
    text = path.read_text(encoding="utf-8")
    entries: list[dict] = []
    current: dict = {}
    in_route_prefixes = False
    in_entry = False

    for line in text.splitlines():
        stripped = line.strip()

        if stripped.startswith("route_prefixes:"):
            in_route_prefixes = True
            continue
        if in_route_prefixes and stripped.startswith("approved_python_exceptions:"):
            break
        if in_route_prefixes and stripped.startswith("django_baseline_routes:"):
            break
        if not in_route_prefixes:
            continue

        # New entry
        if stripped.startswith("- prefix: "):
            if current:
                entries.append(current)
            current = {"prefix": stripped.split("- prefix: ", 1)[1].strip()}
            in_entry = True
            continue
        if stripped.startswith("- "):
            continue

        if in_entry and ":" in stripped:
            k, v = stripped.split(":", 1)
            k = k.strip()
            v = v.strip()
            current[k] = v

    if current:
        entries.append(current)
    return entries


def parse_routes_yaml(path: Path) -> list[str]:
    """Extract all URI patterns from routes.yaml."""
    text = path.read_text(encoding="utf-8")
    uris: list[str] = []

    for line in text.splitlines():
        stripped = line.strip()
        # Single uri
        if stripped.startswith("uri: "):
            uris.append(stripped.split("uri: ", 1)[1].strip())
        # List uri
        if stripped.startswith("- ") and ("*" in stripped or "/api/" in stripped):
            item = stripped[2:].strip()
            uris.append(item)

    return uris


def _strip_wildcard(prefix: str) -> str:
    """Normalize a route prefix for comparison: strip trailing /* or *."""
    p = prefix.strip().rstrip("/")
    p = p.replace("/*", "").replace("*", "")
    return p


def _normalize_uri(uri: str) -> str:
    """Normalize a routes.yaml URI for comparison."""
    u = uri.strip().strip('"').strip("'").rstrip("/")
    u = u.replace("/*", "").replace("*", "")
    return u


def check(ownership_path: Path, routes_path: Path) -> list[str]:
    errors: list[str] = []
    entries = parse_ownership_yaml(ownership_path)

    go_entries = [e for e in entries if e.get("status") == "go"]
    if not go_entries:
        return errors

    uris = parse_routes_yaml(routes_path)
    # Also extract uris lists (the - /api/...* lines under uris: blocks)
    # Re-parse more thoroughly
    text = routes_path.read_text(encoding="utf-8")
    # Find all lines that look like URI patterns
    import re
    uri_patterns = set()
    for line in text.splitlines():
        stripped = line.strip()
        # Match lines like: - /api/tenant/*/...
        if re.match(r'^-\s+/api/', stripped):
            uri_patterns.add(stripped[2:].strip())
        if stripped.startswith("uri: "):
            uri_patterns.add(stripped.split("uri: ", 1)[1].strip())

    normalized_routes = {_normalize_uri(u) for u in uri_patterns}

    for entry in go_entries:
        prefix = entry.get("prefix", "")
        if not prefix:
            continue
        normalized_prefix = _strip_wildcard(prefix)

        # Check if any route covers this prefix
        covered = False
        for route_uri in normalized_routes:
            # Direct prefix match
            if route_uri.startswith(normalized_prefix):
                covered = True
                break
            # Check if route glob would match the prefix
            if normalized_prefix.startswith(route_uri):
                covered = True
                break

        if not covered:
            target = entry.get("target_owner", "unknown")
            note = entry.get("note", "")
            errors.append(
                f"ownership 已登记 Go 但 gateway routes 未找到: "
                f"prefix={prefix} target={target}"
                f"{' — ' + note if note else ''}"
            )

    return errors


def main() -> int:
    parser = argparse.ArgumentParser(
        description="校验 route ownership vs gateway routes 一致性"
    )
    parser.add_argument("--repo-root", help="仓库根目录")
    args = parser.parse_args()

    root = Path(args.repo_root) if args.repo_root else find_repo_root()

    ownership_path = root / "db" / "api_route_ownership.yaml"
    routes_path = root / "taskGateway" / "routes" / "routes.yaml"

    if not ownership_path.exists():
        print(f"错误：找不到 ownership 文件: {ownership_path}", file=sys.stderr)
        return 1
    if not routes_path.exists():
        print(f"错误：找不到 routes 文件: {routes_path}", file=sys.stderr)
        return 1

    errors = check(ownership_path, routes_path)

    if errors:
        print("=== Route Ownership vs Gateway 门禁失败 ===", file=sys.stderr)
        for line in errors:
            print(f"  - {line}", file=sys.stderr)
        print(
            f"\n共 {len(errors)} 条 Go 路由在 ownership 中登记但未在 gateway routes.yaml 中找到。",
            file=sys.stderr,
        )
        return 1

    go_count = len([e for e in parse_ownership_yaml(ownership_path) if e.get("status") == "go"])
    print(f"Route Ownership vs Gateway 门禁通过（{go_count} 条 Go 路由已登记）。")
    return 0


if __name__ == "__main__":
    sys.exit(main())
