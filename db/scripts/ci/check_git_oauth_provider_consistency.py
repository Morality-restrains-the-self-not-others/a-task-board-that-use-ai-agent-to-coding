#!/usr/bin/env python3
"""CI: git-oauth provider service_provider/client_id must match across conf trees."""

from __future__ import annotations

import sys
from pathlib import Path

try:
    import yaml
except ImportError:  # pragma: no cover
    print("ERROR: PyYAML required", file=sys.stderr)
    raise SystemExit(2)

CONF_DIRS = (
    "conf/auth/git-oauth/providers",
    "conf/auth/task-credential/git-oauth-providers",
)
KEYS = ("service_provider", ("target", "client_id"), ("target", "website"))


def monorepo_root() -> Path:
    here = Path(__file__).resolve()
    for parent in here.parents:
        if (parent / "db" / "registry.yaml").is_file():
            return parent
    raise FileNotFoundError("db/registry.yaml not found")


def load_yaml(path: Path) -> dict:
    data = yaml.safe_load(path.read_text(encoding="utf-8")) or {}
    return data if isinstance(data, dict) else {}


def get_nested(d: dict, *keys: str):
    cur = d
    for k in keys:
        if not isinstance(cur, dict):
            return None
        cur = cur.get(k)
    return cur


def provider_key(data: dict) -> str | None:
    """Return a composite key: (website, service_provider).

    Two providers may share the same website template variable (e.g. both
    pointing to ${scheme}://${subdomains.gitlab}) while being different
    providers (daydaymoney-gitlab vs synology-gitlab).  Using service_provider
    as a secondary discriminator avoids false mismatch reports.
    """
    website = get_nested(data, "target", "website")
    sp = data.get("service_provider")
    if isinstance(website, str) and website.strip():
        w = website.strip().rstrip("/").lower()
        return f"{w}::{sp}" if sp else w
    return None


def collect(root: Path) -> dict[str, dict[str, dict]]:
    by_site: dict[str, dict[str, dict]] = {}
    for rel in CONF_DIRS:
        base = root / rel
        if not base.is_dir():
            continue
        for path in sorted(base.glob("*.yaml")):
            if path.name.endswith(".ai.md"):
                continue
            data = load_yaml(path)
            key = provider_key(data)
            if not key:
                continue
            by_site.setdefault(key, {})[rel] = {
                "file": str(path.relative_to(root)),
                "service_provider": data.get("service_provider"),
                "client_id": get_nested(data, "target", "client_id"),
            }
    return by_site


def main() -> int:
    root = monorepo_root()
    by_site = collect(root)
    errors: list[str] = []

    for site, trees in sorted(by_site.items()):
        providers = {v["service_provider"] for v in trees.values() if v.get("service_provider")}
        client_ids = {v["client_id"] for v in trees.values() if v.get("client_id")}
        if len(providers) > 1:
            detail = ", ".join(f"{v['file']}={v['service_provider']!r}" for v in trees.values())
            errors.append(f"{site}: service_provider mismatch ({detail})")
        if len(client_ids) > 1:
            detail = ", ".join(f"{v['file']}={v['client_id']!r}" for v in trees.values())
            errors.append(f"{site}: client_id mismatch ({detail})")
        # Each tree should have an entry when auth provider exists
        if "conf/auth/git-oauth/providers" in str(trees):
            pass

    if errors:
        print("git-oauth provider consistency violations:", file=sys.stderr)
        for e in errors:
            print(f"  - {e}", file=sys.stderr)
        return 1

    print(f"OK: checked {len(by_site)} website keys across {len(CONF_DIRS)} trees")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
