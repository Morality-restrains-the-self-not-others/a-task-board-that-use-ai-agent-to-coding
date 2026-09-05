#!/usr/bin/env python3
"""Build the intended service × URL × API catalog (SSOT overlay for Grafana).

Grafana at :3000 is the *runtime view* (Tempo service map, RED metrics). It is
not the catalog of intended website URLs. This generator joins:

- taskGateway/routes/routes.yaml  → public URI → upstream service
- db/api_route_ownership.yaml     → prefix → owner
- taskFE router pages             → SPA paths

Core vs supporting is declared from value-stream prefixes (not traffic).
Runtime "is this hot?" stays in Grafana HTTP API Latency / Lightweight APM.

Usage:
  python3 db/scripts/ci/build_service_url_api_catalog.py
  python3 db/scripts/ci/build_service_url_api_catalog.py --check
"""

from __future__ import annotations

import argparse
import json
import re
import sys
from collections import Counter
from fnmatch import fnmatch
from pathlib import Path
from typing import Any

from service_url_api_catalog_grafana import (
    DASHBOARD_UID,
    render_grafana_dashboard,
)

try:
    import yaml
except ImportError:  # pragma: no cover
    print("ERROR: PyYAML required (import yaml)", file=sys.stderr)
    raise SystemExit(2)

# Declared core API globs → value stream. Traffic volume is Grafana's job.
CORE_API_GLOBS: tuple[tuple[str, str], ...] = (
    ("/api/auth/*", "user-auth"),
    ("/api/accounts/*", "user-auth"),
    ("/api/kyc/*", "kyc"),
    ("/api/billing/*", "billing"),
    ("/api/tenant/*/billing/*", "billing"),
    ("/api/public/product-pricing/*", "billing"),
    ("/api/public/resource-pricing/*", "billing"),
    ("/api/system-admin/orders/*", "billing"),
    ("/api/system_admin/orders/*", "billing"),
    ("/api/system-admin/profit-sharing/*", "billing"),
    ("/api/system_admin/profit-sharing/*", "billing"),
    ("/api/system-admin/gitlab-regions/*", "billing"),
    ("/api/system_admin/gitlab-regions/*", "billing"),
    ("/api/system-admin/tenant-quotas/*", "billing"),
    ("/api/system_admin/tenant-quotas/*", "billing"),
    ("/api/system-admin/resource-pricing/*", "billing"),
    ("/api/system-admin/user-recharge-consumption/*", "billing"),
    ("/api/system-admin/refund-applications/*", "billing"),
    ("/api/system-admin/invoice-applications/*", "billing"),
    ("/api/system-admin/feedback-link-groups/*", "billing"),
    ("/api/system-admin/feedback-resource-kinds/*", "billing"),
    ("/api/system-admin/refund-policy/*", "billing"),
    ("/api/system-admin/order-number/*", "billing"),
    ("/api/system-admin/idempotency-records/*", "billing"),
    ("/api/system_admin/idempotency-records/*", "billing"),
    ("/api/system-admin/users/*/recharges/*", "billing"),
    ("/api/system_admin/users/*/recharges/*", "billing"),
    ("/api/referral/*", "referral"),
    ("/api/system-admin/referral/*", "referral"),
    ("/api/system-admin/users/*/referral-performance/*", "referral"),
    ("/api/user/*/profile/referral-stats*", "referral"),
    ("/api/tenant/*/projects/*", "project"),
    ("/api/tenant/*/daydaymoney/*", "project"),
    ("/api/projects/*", "project"),
    ("/api/workspace/*", "project"),
    ("/api/tenant/*/workspace/*", "project"),
    ("/api/tenant/*/tasks/*", "task"),
    ("/api/tasks/*", "task"),
    ("/api/sse/*", "task"),
    ("/api/container/*", "compute"),
    ("/api/cloud/*", "compute"),
    ("/api/token/*", "compute"),
    ("/api/system-admin/cloud/*", "compute"),
    ("/api/system-admin/sub-token-providers/*", "compute"),
    ("/api/sub-token-providers/*", "compute"),
    ("/api/system-admin/recommended-llm-providers/*", "compute"),
    ("/api/recommended-llm-providers/*", "compute"),
    ("/api/system-admin/step-full-cos/*", "compute"),
    ("/callback/cloudplatform/*", "compute"),
    ("/api/git-oauth/*", "git-oauth"),
    ("/api/tenant/*/accounts/*", "tenant"),
    ("/api/tenant/*/gitlab-oidc-sso/*", "user-auth"),
)

OPS_MARKERS = ("/health", "/swagger", "/schema", "/metrics", "/gateway/")
CORE_FE_MARKERS: tuple[tuple[str, str], ...] = (
    ("login", "user-auth"),
    ("register", "user-auth"),
    ("onboarding", "user-auth"),
    ("work-panel", "task"),
    ("work_panel", "task"),
    ("billing", "billing"),
    ("order", "billing"),
    ("pricing", "billing"),
    ("project", "project"),
    ("workspace", "project"),
    ("task", "task"),
    ("kyc", "kyc"),
    ("referral", "referral"),
)

PATH_RE = re.compile(r"path:\s*['\"]([^'\"]+)['\"]")
NAME_RE = re.compile(r"name:\s*['\"]([^'\"]+)['\"]")


def monorepo_root() -> Path:
    here = Path(__file__).resolve()
    for parent in here.parents:
        if (parent / "db" / "registry.yaml").is_file():
            return parent
    raise FileNotFoundError("db/registry.yaml not found above catalog builder")


def _glob_hit(uri: str, pattern: str) -> bool:
    u = (uri or "").strip()
    p = (pattern or "").strip()
    if not u or not p or u in ("/", "/*"):
        return False
    if fnmatch(u, p) or fnmatch(p, u):
        return True
    if p.endswith("/*"):
        prefix = p[:-1]  # keep trailing slash, e.g. /api/auth/
        if len(prefix) < 6:
            return False
        return u.startswith(prefix) or u.rstrip("/*").startswith(prefix.rstrip("/"))
    return False


def classify_api(uri: str, upstream: str | None, auth_mode: str | None) -> tuple[str, str | None]:
    mode = (auth_mode or "").strip().lower()
    ups = (upstream or "").strip()
    if mode == "deny" or ups in ("", "null-upstream"):
        return "orphaned", None
    low = uri.lower()
    if any(m in low for m in OPS_MARKERS):
        return "ops", None
    if mode == "machine" or low.startswith("/api/internal/"):
        return "internal", None
    for glob, stream in CORE_API_GLOBS:
        if _glob_hit(uri, glob):
            return "core", stream
    return "supporting", None


def classify_fe(path: str, name: str) -> tuple[str, str | None]:
    blob = f"{path} {name}".lower()
    if "admin" in blob or path.startswith("/system-admin"):
        return "ops", None
    for marker, stream in CORE_FE_MARKERS:
        if marker in blob:
            return "core", stream
    return "supporting", None


def load_gateway_routes(path: Path) -> list[dict[str, Any]]:
    data = yaml.safe_load(path.read_text(encoding="utf-8")) or {}
    rows: list[dict[str, Any]] = []
    for raw in data.get("routes") or []:
        if not isinstance(raw, dict):
            continue
        uris = raw.get("uris") or raw.get("uri") or []
        if isinstance(uris, str):
            uris = [uris]
        rows.append(
            {
                "id": str(raw.get("id") or ""),
                "uris": [str(u) for u in uris],
                "upstream": raw.get("upstream"),
                "auth_mode": raw.get("auth_mode"),
                "priority": raw.get("priority"),
            }
        )
    return rows


def load_ownership(path: Path) -> list[dict[str, str]]:
    data = yaml.safe_load(path.read_text(encoding="utf-8")) or {}
    out: list[dict[str, str]] = []
    for raw in data.get("route_prefixes") or []:
        if not isinstance(raw, dict):
            continue
        prefix = str(raw.get("prefix") or "")
        if not prefix:
            continue
        out.append(
            {
                "prefix": prefix,
                "target_owner": str(raw.get("target_owner") or ""),
                "status": str(raw.get("status") or ""),
            }
        )
    return out


def match_owner(uri: str, entries: list[dict[str, str]]) -> dict[str, str]:
    best: dict[str, str] | None = None
    best_len = -1
    for e in entries:
        if _glob_hit(uri, e["prefix"] if e["prefix"].endswith("*") else e["prefix"].rstrip("/") + "/*"):
            n = len(e["prefix"].replace("*", ""))
            if n > best_len:
                best = e
                best_len = n
    return best or {"prefix": "", "target_owner": "", "status": ""}


def load_fe_pages(files: list[Path]) -> list[dict[str, str]]:
    pages: list[dict[str, str]] = []
    for f in files:
        if not f.is_file():
            continue
        text = f.read_text(encoding="utf-8")
        # Pair path with the nearest following name in the same object-ish window.
        for m in PATH_RE.finditer(text):
            path = m.group(1)
            window = text[m.end() : m.end() + 240]
            nm = NAME_RE.search(window)
            pages.append({"path": path, "name": nm.group(1) if nm else "", "source": f.name})
    return pages


def build_catalog(
    routes_path: Path,
    ownership_path: Path,
    fe_files: list[Path],
) -> dict[str, Any]:
    ownership = load_ownership(ownership_path)
    api_routes: list[dict[str, Any]] = []
    by_service: dict[str, Counter[str]] = {}
    for row in load_gateway_routes(routes_path):
        for uri in row["uris"]:
            role, stream = classify_api(uri, row.get("upstream"), row.get("auth_mode"))
            owner = match_owner(uri, ownership)
            svc = str(row.get("upstream") or "(none)")
            by_service.setdefault(svc, Counter())[role] += 1
            api_routes.append(
                {
                    "id": row["id"],
                    "uri": uri,
                    "upstream": row.get("upstream"),
                    "auth_mode": row.get("auth_mode") or "",
                    "priority": row.get("priority"),
                    "owner": owner["target_owner"],
                    "owner_status": owner["status"],
                    "role": role,
                    "value_stream": stream,
                }
            )
    fe_pages = []
    for page in load_fe_pages(fe_files):
        role, stream = classify_fe(page["path"], page["name"])
        fe_pages.append({**page, "role": role, "value_stream": stream})
    summary = {
        "api_routes": len(api_routes),
        "fe_pages": len(fe_pages),
        "core_api": sum(1 for r in api_routes if r["role"] == "core"),
        "supporting_api": sum(1 for r in api_routes if r["role"] == "supporting"),
        "ops_api": sum(1 for r in api_routes if r["role"] == "ops"),
        "internal_api": sum(1 for r in api_routes if r["role"] == "internal"),
        "orphaned_api": sum(1 for r in api_routes if r["role"] == "orphaned"),
        "core_fe": sum(1 for p in fe_pages if p["role"] == "core"),
        "services": len([k for k in by_service if k not in ("(none)", "null-upstream")]),
    }
    return {
        "title": "Service URL / API catalog",
        "decision": "grafana-is-view-not-ssot",
        "grafana_url": "http://10.2.150.68:3000/d/service-url-api-catalog/service-url-api-catalog",
        "generated_from": [
            "taskGateway/routes/routes.yaml",
            "db/api_route_ownership.yaml",
            "taskFE/app/src/router/{public,tenant,admin}Routes.js",
        ],
        "summary": summary,
        "by_service": {k: dict(v) for k, v in sorted(by_service.items())},
        "api_routes": api_routes,
        "fe_pages": fe_pages,
    }


def render_markdown(catalog: dict[str, Any]) -> str:
    s = catalog.get("summary") or {}
    lines = [
        "# Service URL / API catalog",
        "",
        "> Intended topology (config SSOT). Grafana `:3000` overlays **runtime**",
        "> traffic and Tempo service map — it is not the source of website URLs.",
        "",
        f"- Grafana dashboard: `{catalog.get('grafana_url', DASHBOARD_UID)}`",
        (
            f"- API routes: **{s.get('api_routes', len(catalog.get('api_routes') or []))}** "
            f"(core {s.get('core_api', '?')}, supporting {s.get('supporting_api', '?')}, "
            f"ops {s.get('ops_api', '?')}, internal {s.get('internal_api', '?')}, "
            f"orphaned {s.get('orphaned_api', '?')})"
        ),
        (
            f"- SPA pages: **{s.get('fe_pages', len(catalog.get('fe_pages') or []))}** "
            f"(core {s.get('core_fe', '?')})"
        ),
        "",
        "## How to read roles",
        "",
        "| role | meaning |",
        "|---|---|",
        "| core | Declared value-stream path (auth/billing/task/project/compute/kyc) |",
        "| supporting | Needed but not a primary user journey |",
        "| ops | Health, swagger, schema, admin chrome |",
        "| internal | Machine/`/api/internal` — not a browser URL |",
        "| orphaned | deny or null-upstream (dead public path) |",
        "",
        "## By upstream service",
        "",
        "| service | core | supporting | ops | internal | orphaned |",
        "|---|---:|---:|---:|---:|---:|",
    ]
    by = catalog.get("by_service") or {}
    for svc, counts in by.items():
        lines.append(
            f"| `{svc}` | {counts.get('core', 0)} | {counts.get('supporting', 0)} | "
            f"{counts.get('ops', 0)} | {counts.get('internal', 0)} | {counts.get('orphaned', 0)} |"
        )
    lines += ["", "## Core API routes", "", "| uri | upstream | owner | stream | auth |", "|---|---|---|---|---|"]
    for r in catalog.get("api_routes") or []:
        if r.get("role") != "core":
            continue
        lines.append(
            f"| `{r.get('uri')}` | `{r.get('upstream')}` | `{r.get('owner') or ''}` | "
            f"{r.get('value_stream') or ''} | {r.get('auth_mode') or ''} |"
        )
    lines += ["", "## Orphaned / deny routes", "", "| uri | id | upstream | auth |", "|---|---|---|---|"]
    orphans = [r for r in (catalog.get("api_routes") or []) if r.get("role") == "orphaned"]
    if not orphans:
        lines.append("| _(none)_ | | | |")
    for r in orphans:
        lines.append(
            f"| `{r.get('uri')}` | `{r.get('id')}` | `{r.get('upstream')}` | {r.get('auth_mode') or ''} |"
        )
    lines += ["", "## Core SPA pages", "", "| path | name | stream | source |", "|---|---|---|---|"]
    for p in catalog.get("fe_pages") or []:
        if p.get("role") != "core":
            continue
        lines.append(
            f"| `{p.get('path')}` | {p.get('name') or ''} | {p.get('value_stream') or ''} | `{p.get('source')}` |"
        )
    lines += [
        "",
        "## Regenerate",
        "",
        "```bash",
        "python3 db/scripts/ci/build_service_url_api_catalog.py",
        "python3 db/scripts/ci/build_service_url_api_catalog.py --check",
        "```",
        "",
    ]
    return "\n".join(lines) + "\n"


def default_paths(root: Path) -> dict[str, Path]:
    return {
        "routes": root / "taskGateway" / "routes" / "routes.yaml",
        "ownership": root / "db" / "api_route_ownership.yaml",
        "json_out": root / "docs" / "architecture" / "service-url-api-catalog.json",
        "md_out": root / "docs" / "architecture" / "service-url-api-catalog.md",
        "dash_out": root
        / "AiMonitor"
        / "grafana"
        / "provisioning"
        / "dashboards"
        / "files"
        / "service-url-api-catalog.json",
    }


def fe_router_files(root: Path) -> list[Path]:
    d = root / "taskFE" / "app" / "src" / "router"
    return [d / n for n in ("publicRoutes.js", "tenantRoutes.js", "adminRoutes.js")]


def write_outputs(catalog: dict[str, Any], paths: dict[str, Path]) -> None:
    paths["json_out"].parent.mkdir(parents=True, exist_ok=True)
    paths["json_out"].write_text(json.dumps(catalog, ensure_ascii=False, indent=2) + "\n", encoding="utf-8")
    paths["md_out"].write_text(render_markdown(catalog), encoding="utf-8")
    dash = render_grafana_dashboard(catalog)
    paths["dash_out"].parent.mkdir(parents=True, exist_ok=True)
    paths["dash_out"].write_text(json.dumps(dash, ensure_ascii=False, indent=2) + "\n", encoding="utf-8")


def check_outputs(catalog: dict[str, Any], paths: dict[str, Path]) -> list[str]:
    errors: list[str] = []
    expected = {
        paths["json_out"]: json.dumps(catalog, ensure_ascii=False, indent=2) + "\n",
        paths["md_out"]: render_markdown(catalog),
        paths["dash_out"]: json.dumps(render_grafana_dashboard(catalog), ensure_ascii=False, indent=2) + "\n",
    }
    for path, body in expected.items():
        if not path.is_file():
            errors.append(f"missing {path}")
            continue
        if path.read_text(encoding="utf-8") != body:
            errors.append(f"stale {path.name}")
    return errors


def main(argv: list[str] | None = None) -> int:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--check", action="store_true", help="fail if generated files are stale")
    parser.add_argument("--repo-root", type=Path, default=None)
    args = parser.parse_args(argv)
    root = args.repo_root or monorepo_root()
    paths = default_paths(root)
    catalog = build_catalog(paths["routes"], paths["ownership"], fe_router_files(root))
    if args.check:
        errs = check_outputs(catalog, paths)
        if errs:
            print("VIOLATION: service URL/API catalog stale:", file=sys.stderr)
            for e in errs:
                print(f"  - {e}", file=sys.stderr)
            print("Run: python3 db/scripts/ci/build_service_url_api_catalog.py", file=sys.stderr)
            return 1
        print("OK: service URL/API catalog up to date")
        return 0
    write_outputs(catalog, paths)
    s = catalog["summary"]
    print(
        f"Wrote catalog: api={s['api_routes']} core_api={s['core_api']} "
        f"fe={s['fe_pages']} orphaned={s['orphaned_api']}"
    )
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
