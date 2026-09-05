#!/usr/bin/env python3
"""check_go_routes_vs_apisix.py — Cross-check Go service handler routes vs APISIX routes.yaml

OPT-20260728-014: Prevent silent 404s when Go services add new API endpoints
but the gateway routes.yaml is not updated. Scans ALL Go services, not just taskBill.

Usage:
    python3 scripts/ci/check_go_routes_vs_apisix.py [--ci] [--service taskAuth]
    --ci        Exit non-zero on missing routes (for CI pipeline)
    --service   Only check a specific service (default: all Go services)
"""

from __future__ import annotations

import argparse
import os
import re
import sys
from collections import defaultdict
from pathlib import Path
from typing import Dict, List, Optional, Set, Tuple

# ---------------------------------------------------------------------------
# Configuration
# ---------------------------------------------------------------------------

REPO_ROOT = Path(__file__).resolve().parent.parent.parent.parent
ROUTES_FILE = REPO_ROOT / "taskGateway" / "routes" / "routes.yaml"

# Service name -> list of (source_file, relative_to_repo_root)
GO_SERVICES: Dict[str, List[Tuple[str, str]]] = {
    "taskAuth": [
        ("taskAuth/src/handlers.go", "taskAuth/src/handlers.go"),
    ],
    "taskReferral": [
        ("taskReferral/src/handlers.go", "taskReferral/src/handlers.go"),
    ],
    "taskBill": [
        ("taskBill/src/handlers.go", "taskBill/src/handlers.go"),
    ],
    "taskCredentialService": [
        ("taskCredentialService/interfaces/handlers.go", "taskCredentialService/interfaces/handlers.go"),
    ],
    "taskProjectService": [
        ("taskProjectService/src/main.go", "taskProjectService/src/main.go"),
    ],
    "taskTenantService": [
        ("taskTenantService/src/main.go", "taskTenantService/src/main.go"),
    ],
    "taskTaskService": [
        ("taskTaskService/src/main.go", "taskTaskService/src/main.go"),
    ],
    "taskCloudService": [
        ("taskCloudService/src/main.go", "taskCloudService/src/main.go"),
    ],
    "taskContainerGateway": [
        ("taskContainerGateway/src/handlers.go", "taskContainerGateway/src/handlers.go"),
    ],
    "taskAgentSupport": [
        ("taskAgentSupport/src/handlers.go", "taskAgentSupport/src/handlers.go"),
    ],
    "taskAIEndPoint": [
        ("taskAIEndPoint/src/handlers.go", "taskAIEndPoint/src/handlers.go"),
    ],
}

# Paths intentionally NOT exposed through public gateway:
#   /api/internal/*     — inter-service RPC (explicitly denied: priority 1000)
#   /api/health*        — health check (monitoring uses separate port or /api/health/* → django)
#   /api/schema*        — OpenAPI schema (internal/docs only)
#   /api/swagger*       — Swagger UI (internal/docs only)
#   /api/billing/wechat/notify*  — WeChat callback (direct IP whitelist)
#   /api/billing/profitsharing/notify* — WeChat profit-sharing callback
#   /metrics            — Prometheus metrics (scraped directly)
#   /callback/*         — OAuth callbacks (public, not token-authed)
#   /.well-known/*      — OIDC discovery (public)
#   /v1/*               — legacy credential service endpoints (internal)
SKIPPABLE_EXACT: Tuple[str, ...] = (
    "/api/tenant",   # Bare sub-router mount point; real routes are /api/tenant/*/...
    "/api/tenant_id",  # Bare sub-router mount point (taskTaskService handleTenantIDPrefixedRoutes);
                      # real routes are /api/tenant_id/*/workspaceId/*/tasks/*/comments/* etc.
)

SKIPPABLE_PREFIXES: Tuple[str, ...] = (
    "/api/internal/",
    "/api/internal",
    "/api/health",
    "/api/schema",
    "/api/swagger",
    "/api/schema-internal",
    "/api/swagger-internal",
    "/api/billing/wechat/notify",
    "/api/billing/profitsharing/notify",
    "/api/live",
    "/api/metrics",
    "/api/vendor/",
    "/api/vendor",
    "/metrics",
    "/health",
    "/callback/",
    "/.well-known/",
    "/v1/",
)

# Route registration patterns in Go code
# Pattern 1: mux.HandleFunc("GET /path", handler)  — Go 1.22+ method-prefixed
# Pattern 2: mux.HandleFunc("/path", handler)       — unprefixed
# Pattern 3: h.RegisterRoutes(mux) with mux.HandleFunc("/path", ...)
#
# Group 1 captures the path only (stripping method prefix if present)
RE_HANDLE_FUNC = re.compile(
    r'''HandleFunc\(\s*"(?:GET|POST|PUT|PATCH|DELETE|HEAD|OPTIONS)\s+'''
    r'''((?:/[^"{}\s]*|(?:/\{[^}]+\}))+/?\**)'''
    r'''"'''
)
RE_HANDLE_FUNC_NO_METHOD = re.compile(
    r'''HandleFunc\(\s*"'''
    r'''((?:/[^"{}\s]*|(?:/\{[^}]+\}))+/?\**)'''
    r'''"'''
)

# Pattern 4: parts-based dispatch inside inline handler closures (OPT-20260824-072).
#   mux.HandleFunc("/api/tenant/", func(w http.ResponseWriter, r *http.Request) {
#       parts := cleanPath(r, "/api/tenant/")
#       if len(parts) >= 4 && parts[1] == "workspace" && parts[3] == "todos" {
#           ...
#       }
#   })
# A literal comparison `parts[N] == "literal"` at index N means the N-th path
# segment after the mount prefix is fixed to `literal`; indices that are only
# read as variables (e.g. parts[2] workspace id) are wildcards `*`.
RE_HANDLE_FUNC_DISPATCH = re.compile(
    r'''HandleFunc\(\s*"(?P<prefix>[^"]+)"\s*,\s*func\s*\([^)]*\)\s*\{'''
)
RE_PARTS_EQ_LITERAL = re.compile(
    r'''parts\[\s*(\d+)\s*\]\s*==\s*"([A-Za-z0-9][A-Za-z0-9._-]*)"'''
)

# Synthesized dispatch routes intentionally NOT exposed through the gateway as
# their literal `/api/tenant/` form — either retired or served via a convention
# path that IS gateway-routed. Excluding avoids false MISSING noise while keeping
# the extraction generic for newly added parts-dispatch routes.
DISPATCH_ROUTE_EXCLUSIONS: Tuple[str, ...] = (
    "/api/tenant/*/projects/translate-branch-title",  # retired on TTS (501); owner task-project-service
    "/api/tenant/*/tasks/search",  # 租户级搜索经约定路径 /api/tasks/search/tenant_id/... 走网关；/api/tenant/ 形态被 tenant-service(863) 兜底
    "/api/tenant/*/tasks/*/comments",  # 评论 CRUD 经约定路径 /api/tasks/{taskId}/comments/tenant_id/...；/api/tenant/ 形态被 863 兜底
)

# ---------------------------------------------------------------------------
# Route extraction
# ---------------------------------------------------------------------------


def extract_go_routes(file_path: Path) -> Set[str]:
    """Extract all unique route paths registered in a Go handler file.

    Returns normalized paths (no trailing slash, Go {param} → *, sorted, deduped).
    Skips catch-all root handler ("/").
    """
    if not file_path.exists():
        return set()

    content = file_path.read_text(encoding="utf-8", errors="replace")
    routes: Set[str] = set()

    # Method-prefixed handlers (Go 1.22+): "GET /api/path", "POST /api/path/{id}/"
    for match in RE_HANDLE_FUNC.finditer(content):
        path = match.group(1)
        normalized = _normalize_go_path(path)
        if normalized and normalized != "/":
            routes.add(normalized)

    # Unprefixed handlers: "/api/path", "/api/path/"
    for match in RE_HANDLE_FUNC_NO_METHOD.finditer(content):
        path = match.group(1)
        normalized = _normalize_go_path(path)
        if normalized and normalized != "/":
            routes.add(normalized)

    # Parts-dispatch routes (Pattern 4): synthesize from inline closures.
    routes |= _extract_dispatch_routes(content)

    return routes


def _find_handler_body_end(content: str, func_start: int) -> Optional[int]:
    """Return index of the closing '}' of an inline func closure body.

    `func_start` points at the 'func' keyword (or anywhere before the opening
    brace). Balances braces while skipping over string literals.
    """
    brace = content.find("{", func_start)
    if brace == -1:
        return None
    depth = 0
    in_str = False
    i = brace
    while i < len(content):
        ch = content[i]
        if in_str:
            if ch == '"' and content[i - 1] != "\\":
                in_str = False
            elif ch == "\\":
                i += 1
        else:
            if ch == '"':
                in_str = True
            elif ch == "{":
                depth += 1
            elif ch == "}":
                depth -= 1
                if depth == 0:
                    return i
        i += 1
    return None


def _synthesize_dispatch_path(prefix: str, literals: Dict[int, str]) -> str:
    """Build `/prefix/*/lit/...` from a mount prefix and parts-index literals.

    indices 0..max_index contribute `literal` where known, `*` otherwise.
    """
    segments = [prefix.rstrip("/")]
    max_idx = max(literals)
    for i in range(max_idx + 1):
        segments.append(literals.get(i, "*"))
    return "/".join(segments)


def _extract_dispatch_routes(content: str) -> Set[str]:
    """Synthesize routes from `parts[N] == "literal"` dispatch in inline closures.

    OPT-20260824-072: `check_go_routes_vs_apisix` previously only saw literal
    `mux.HandleFunc("...")` registrations. taskTaskService routes like
    `/api/tenant/*/workspace/*/todos` and `.../queue-schedule` are dispatched
    inside the `/api/tenant/` handler via `parts[1] == "workspace"` style checks
    and were invisible to the checker — so a dropped gateway route went
    undetected until a live 404. This extracts those dispatch patterns.
    """
    routes: Set[str] = set()
    for match in RE_HANDLE_FUNC_DISPATCH.finditer(content):
        prefix = match.group("prefix")
        # The regex consumed the opening brace; start balancing at its position
        # (match.end() - 1) so the handler body depth is counted correctly.
        body_end = _find_handler_body_end(content, match.end() - 1)
        if body_end is None:
            continue
        body = content[match.end():body_end]
        for line in body.splitlines():
            stripped = line.strip()
            # Only treat `if`-condition lines as dispatch branches. This skips
            # `parts` comparisons that appear in variable assignments, error
            # strings, or switch-case expressions with a different semantic.
            if not (stripped.startswith("if ") or stripped.startswith("} else if ")):
                continue
            literals = {int(k): v for k, v in RE_PARTS_EQ_LITERAL.findall(line)}
            if not literals:
                continue
            path = _synthesize_dispatch_path(prefix, literals)
            if path in DISPATCH_ROUTE_EXCLUSIONS:
                continue
            normalized = _normalize_go_path(path)
            if normalized and normalized != "/":
                routes.add(normalized)
    return routes


def _normalize_go_path(path: str) -> str:
    """Normalize a Go route path for comparison with APISIX URIs.

    - Strip trailing slash
    - Convert Go {param} placeholders to *
    - Strip trailing wildcard segment
    """
    path = path.rstrip("/")
    if not path:
        return ""
    # Replace Go template parameters {xxx} with *
    path = re.sub(r'/\{[^}]+\}', '/*', path)
    # Remove trailing /* if present (prefix matching)
    if path.endswith("/*"):
        path = path[:-2]
    return path


def is_skippable(path: str) -> bool:
    """Check if a path is intentionally not exposed through the gateway."""
    if path in SKIPPABLE_EXACT:
        return True
    for prefix in SKIPPABLE_PREFIXES:
        if path.startswith(prefix):
            return True
    return False


def _strip_apisix_wildcard(uri: str) -> str:
    """Strip APISIX wildcards from a URI pattern for matching purposes.

    APISIX:
      - '*' matches exactly one path segment (not including '/')
      - '**' is NOT a valid APISIX wildcard
      - '/*' as a suffix matches one segment under the prefix
    """
    uri = uri.rstrip("/")
    # Remove APISIX single-segment wildcards
    while uri.endswith("/*"):
        uri = uri[:-2]
    # NOTE: mid-path wildcards (e.g., /api/tenant/*/workspaces) are intentionally PRESERVED
    # as a literal '*' segment — the Go side normalizes {param} → '*' the same way, so
    # /api/system-admin/users/*/recharges (gateway) matches /api/system-admin/users/*/recharges (Go).
    # Collapsing '/*/' → '/' here broke that match and produced false "MISSING" reports
    # for every mid-path wildcard route (see OPT-20260806-022).
    return uri


def extract_gateway_routes(routes_file: Path) -> Dict[str, Set[str]]:
    """Parse routes.yaml and return {upstream_name: set_of_normalized_uri_prefixes}."""
    try:
        import yaml
    except ImportError:
        print("WARNING: PyYAML not installed. Install with: pip install pyyaml", file=sys.stderr)
        return {}

    if not routes_file.exists():
        print(f"WARNING: routes file not found: {routes_file}", file=sys.stderr)
        return {}

    with open(routes_file, "r", encoding="utf-8") as f:
        doc = yaml.safe_load(f)

    upstream_routes: Dict[str, Set[str]] = defaultdict(set)

    for route in doc.get("routes", []):
        upstream = route.get("upstream", "")
        if not upstream:
            continue

        # Collect all URI patterns (uri or uris)
        uri_patterns: List[str] = []
        for key in ("uri", "uris"):
            vals = route.get(key)
            if not vals:
                continue
            if isinstance(vals, str):
                uri_patterns.append(vals)
            elif isinstance(vals, list):
                uri_patterns.extend(str(v) for v in vals)

        for uri in uri_patterns:
            uri = uri.strip()
            if not uri or uri == "/*":
                continue
            normalized = _strip_apisix_wildcard(uri)
            if normalized:
                upstream_routes[upstream].add(normalized)

    return dict(upstream_routes)


# ---------------------------------------------------------------------------
# Coverage checking
# ---------------------------------------------------------------------------


def _path_covered(go_path: str, gateway_prefixes: Set[str]) -> bool:
    """Check if a Go handler path is covered by any gateway route prefix.

    A Go path /a/b/c is covered if the gateway has:
      - exact match: /a/b/c
      - prefix match: /a/b (with /* wildcard in original)
    """
    # Exact match
    if go_path in gateway_prefixes:
        return True

    # Prefix match: walk up the path tree
    parts = go_path.strip("/").split("/")
    for i in range(len(parts), 0, -1):
        prefix = "/" + "/".join(parts[:i])
        if prefix in gateway_prefixes:
            return True

    return False


def check_service(
    service_name: str,
    handler_files: List[Tuple[str, str]],
    gateway_routes: Dict[str, Set[str]],
) -> Tuple[int, int, int, List[str]]:
    """Check one Go service's route coverage.

    Returns: (checked, skipped, missing, missing_paths)
    """
    go_routes: Set[str] = set()
    for rel_path, _ in handler_files:
        file_path = REPO_ROOT / rel_path
        go_routes |= extract_go_routes(file_path)

    gw_prefixes = gateway_routes.get(service_name, set())

    checked = 0
    skipped = 0
    missing_paths: List[str] = []

    for path in sorted(go_routes):
        if not path or path == "/":
            continue
        if is_skippable(path):
            skipped += 1
            continue
        checked += 1
        if not _path_covered(path, gw_prefixes):
            missing_paths.append(path)

    return checked, skipped, len(missing_paths), missing_paths


# ---------------------------------------------------------------------------
# Main
# ---------------------------------------------------------------------------


def main() -> int:
    parser = argparse.ArgumentParser(description="Cross-check Go handler routes vs APISIX routes.yaml")
    parser.add_argument("--ci", action="store_true", help="Exit non-zero on missing routes")
    parser.add_argument("--service", type=str, default=None, help="Only check a specific service")
    parser.add_argument("--verbose", "-v", action="store_true", help="Show checked and skipped paths")
    args = parser.parse_args()

    print("=== Go Route → APISIX Gateway Route Coverage Check ===")
    print(f"Routes file: {ROUTES_FILE}")
    print()

    gateway_routes = extract_gateway_routes(ROUTES_FILE)
    if not gateway_routes:
        print("ERROR: Could not parse gateway routes (PyYAML missing or file not found).")
        return 2

    services = {args.service: GO_SERVICES[args.service]} if args.service else GO_SERVICES

    total_checked = 0
    total_skipped = 0
    total_missing = 0
    all_missing: Dict[str, List[str]] = {}

    for service_name, handler_files in sorted(services.items()):
        checked, skipped, missing_count, missing_paths = check_service(
            service_name, handler_files, gateway_routes
        )

        if checked == 0 and skipped == 0:
            continue  # Service file not found, skip silently

        total_checked += checked
        total_skipped += skipped
        total_missing += missing_count

        status = "✅" if missing_count == 0 else "❌"
        print(f"{status} {service_name}: {checked} checked, {skipped} skipped, {missing_count} missing")

        if args.verbose:
            for path in sorted(go_routes := set()):
                pass  # Already processed above

        if missing_paths:
            all_missing[service_name] = missing_paths
            for path in missing_paths:
                print(f"   ❌ MISSING: {path}")

    print()
    print(f"--- Summary: {total_checked} checked, {total_skipped} skipped, {total_missing} missing ---")

    if total_missing == 0:
        print("✅ All public Go handler paths covered by gateway routes.")
        return 0
    else:
        print(f"❌ {total_missing} route(s) missing from {ROUTES_FILE}")
        print("   Add routes for the paths listed above to routes.yaml, then run routes-to-apisix.py.")
        if args.ci:
            return 1
        return 0


if __name__ == "__main__":
    sys.exit(main())
