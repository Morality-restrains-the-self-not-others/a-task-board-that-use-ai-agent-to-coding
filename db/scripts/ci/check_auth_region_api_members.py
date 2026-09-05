#!/usr/bin/env python3
"""P2 CI gate: dataMigrate taskAuth api resource members must map to Go HandleFunc.

Pragmatic scope (do NOT require every HandleFunc to have a region member — that
would false-fail hundreds of public/auth routes):

1. Parse dataMigrate/taskAuth/*.sql for auth_resource_member rows with
   member_kind = api and extract member_key ("METHOD /path").
2. Validate member_key format: METHOD + absolute /api/... path.
3. Scan taskAuth + taskTenantService (+ optional extra dirs) for:
   - mux.HandleFunc("METHOD path", ...) / HandleFunc("path", ...)
   - documented METHOD /api/... strings in comments
   - catch-all prefix routers (e.g. /api/tenant/)
4. Fail if an SQL api member_key has no matching registration (prefix OK for
   {param} templates and catch-all routers).

Usage:
  python3 db/scripts/ci/check_auth_region_api_members.py
  python3 db/scripts/ci/check_auth_region_api_members.py --root /path/to/repo

Exit: 0 pass, 1 drift, 2 IO/config error.
"""

from __future__ import annotations

import argparse
import re
import sys
from dataclasses import dataclass
from pathlib import Path

HTTP_METHODS = frozenset({"GET", "POST", "PUT", "PATCH", "DELETE", "HEAD", "OPTIONS"})

MEMBER_KEY_RE = re.compile(
    r"^(GET|POST|PUT|PATCH|DELETE|HEAD|OPTIONS)\s+(/[^\s'\"]+)$"
)

# INSERT ... 'api', 'GET /api/...'  or "api", "GET /api/..."
SQL_API_MEMBER_RE = re.compile(
    r"""['"]api['"]\s*,\s*['"]((?:GET|POST|PUT|PATCH|DELETE|HEAD|OPTIONS)\s+/[^'"]+)['"]""",
    re.IGNORECASE,
)

HANDLE_METHOD_PATH_RE = re.compile(
    r"""HandleFunc\(\s*"(?:(GET|POST|PUT|PATCH|DELETE|HEAD|OPTIONS)\s+)?([^"]+)"\s*,"""
)

# Comment / doc strings: PUT /api/auth/roles/...
DOC_METHOD_PATH_RE = re.compile(
    r"\b(GET|POST|PUT|PATCH|DELETE|HEAD|OPTIONS)\s+(/api/[^\s`*'\"<>]+)"
)

REQUIRE_REGION_RE = re.compile(r"""RequireRegion\(\s*"([^"]+)" """)


@dataclass(frozen=True)
class ApiMember:
    key: str  # "METHOD /path"
    method: str
    path: str
    source: str


def monorepo_root(start: Path | None = None) -> Path:
    here = (start or Path(__file__)).resolve()
    for parent in [here, *here.parents]:
        if (parent / "db" / "registry.yaml").is_file():
            return parent
    raise FileNotFoundError("db/registry.yaml not found above checker")


def _norm_path(path: str) -> str:
    p = path.strip()
    if not p.startswith("/"):
        p = "/" + p
    # Keep trailing slash policy loose: normalize to trailing / unless template-only tip
    if "{" not in p.rstrip("/").split("/")[-1] and not p.endswith("/"):
        p += "/"
    return p


def _strip_query(path: str) -> str:
    return path.split("?", 1)[0].split("#", 1)[0]


def parse_api_members_from_sql(sql_text: str, source: str) -> list[ApiMember]:
    out: list[ApiMember] = []
    for m in SQL_API_MEMBER_RE.finditer(sql_text):
        raw = m.group(1).strip()
        mm = MEMBER_KEY_RE.match(raw)
        if not mm:
            continue
        method, path = mm.group(1).upper(), _norm_path(_strip_query(mm.group(2)))
        out.append(ApiMember(key=f"{method} {path}", method=method, path=path, source=source))
    return out


def validate_member_key_format(raw: str) -> str | None:
    """Return error message if invalid, else None."""
    m = MEMBER_KEY_RE.match(raw.strip())
    if not m:
        return f"invalid api member_key format (want 'METHOD /path'): {raw!r}"
    path = m.group(2)
    if not path.startswith("/api/"):
        return f"api member_key path must start with /api/: {raw!r}"
    return None


def extract_handle_routes(go_text: str) -> set[str]:
    """Return normalized 'METHOD path' or 'ANY path' registrations."""
    routes: set[str] = set()
    for m in HANDLE_METHOD_PATH_RE.finditer(go_text):
        method = (m.group(1) or "ANY").upper()
        path = _norm_path(_strip_query(m.group(2)))
        routes.add(f"{method} {path}")
    for m in DOC_METHOD_PATH_RE.finditer(go_text):
        method = m.group(1).upper()
        path = _norm_path(_strip_query(m.group(2).rstrip(".,;)")))
        if path.startswith("/api/"):
            routes.add(f"{method} {path}")
    return routes


def extract_require_regions(go_text: str) -> set[str]:
    return {m.group(1) for m in REQUIRE_REGION_RE.finditer(go_text)}


def _path_template_prefix(path: str) -> str:
    """Collapse {param} segments for prefix matching."""
    parts = []
    for seg in path.split("/"):
        if not seg:
            continue
        if seg.startswith("{") and seg.endswith("}"):
            parts.append("{}")
        else:
            parts.append(seg)
    return "/" + "/".join(parts) + "/"


def route_covers(member: ApiMember, routes: set[str]) -> bool:
    m_path = _path_template_prefix(member.path)
    m_method = member.method
    for lit in routes:
        parts = lit.split(" ", 1)
        if len(parts) != 2:
            continue
        r_method, r_path = parts[0], _path_template_prefix(parts[1])
        method_ok = r_method == "ANY" or r_method == m_method
        if not method_ok:
            continue
        # Exact / prefix either direction (catch-all /api/tenant/ covers /api/tenant/member-role/)
        if r_path == m_path:
            return True
        if m_path.startswith(r_path) or r_path.startswith(m_path):
            return True
        # Template-aware: /api/auth/roles/role_id/{}/resource-groups/ vs concrete seed
        if _templates_compatible(m_path, r_path):
            return True
    return False


def _templates_compatible(a: str, b: str) -> bool:
    sa, sb = a.strip("/").split("/"), b.strip("/").split("/")
    n = min(len(sa), len(sb))
    for i in range(n):
        if sa[i] == "{}" or sb[i] == "{}":
            continue
        if sa[i] != sb[i]:
            return False
    # Remaining longer side may only be extra concrete segments after a prefix match
    longer, shorter = (sa, sb) if len(sa) >= len(sb) else (sb, sa)
    if len(longer) == len(shorter):
        return True
    # Allow seed path to be a prefix of registered path (or vice versa)
    return True if n == len(shorter) else False


def collect_go_routes(root: Path, dirs: list[str]) -> set[str]:
    routes: set[str] = set()
    for rel in dirs:
        base = root / rel
        if not base.is_dir():
            continue
        for path in base.rglob("*.go"):
            if path.name.endswith("_test.go"):
                # Still allow comments in tests? Prefer production sources only.
                continue
            try:
                text = path.read_text(encoding="utf-8")
            except OSError:
                continue
            routes |= extract_handle_routes(text)
    return routes


def collect_sql_members(root: Path) -> list[ApiMember]:
    migrate = root / "dataMigrate" / "taskAuth"
    if not migrate.is_dir():
        return []
    members: list[ApiMember] = []
    for sql in sorted(migrate.glob("*.sql")):
        text = sql.read_text(encoding="utf-8")
        members.extend(parse_api_members_from_sql(text, str(sql.relative_to(root))))
    # de-dupe by key
    seen: set[str] = set()
    uniq: list[ApiMember] = []
    for m in members:
        if m.key in seen:
            continue
        seen.add(m.key)
        uniq.append(m)
    return uniq


def run_check(root: Path) -> tuple[int, str]:
    members = collect_sql_members(root)
    routes = collect_go_routes(
        root,
        [
            "taskAuth/src",
            "taskTenantService/src",
            "taskProjectService/src",
            "taskCloudService/src",
            "taskBill/src",
            "taskTaskService/src",
        ],
    )

    lines: list[str] = []
    bad_format: list[str] = []
    missing: list[ApiMember] = []

    for m in members:
        err = validate_member_key_format(f"{m.method} {m.path.rstrip('/')}/" if not m.path.endswith("/") else m.key)
        # re-validate original-ish
        fmt_err = validate_member_key_format(f"{m.method} {m.path}")
        if fmt_err:
            bad_format.append(f"{m.source}: {fmt_err}")
            continue
        if not route_covers(m, routes):
            missing.append(m)

    # Also scan RequireRegion keys for informational count (not hard-fail alone)
    region_keys: set[str] = set()
    for rel in ("taskAuth/src", "taskTenantService/src", "shareLib/authz"):
        base = root / rel
        if not base.exists():
            continue
        paths = [base] if base.is_file() else list(base.rglob("*.go"))
        for path in paths:
            if path.suffix != ".go":
                continue
            try:
                region_keys |= extract_require_regions(path.read_text(encoding="utf-8"))
            except OSError:
                continue

    if bad_format:
        lines.append("INVALID api member_key format:")
        lines.extend(f"  - {x}" for x in bad_format)
    if missing:
        lines.append("SQL api member_key has no matching HandleFunc (or documented route):")
        for m in missing:
            lines.append(f"  - {m.key}  ({m.source})")

    if bad_format or missing:
        lines.append(
            f"Checked {len(members)} api members against {len(routes)} Go routes; "
            f"RequireRegion keys seen: {len(region_keys)}"
        )
        return 1, "\n".join(lines) + "\n"

    return 0, (
        f"OK auth region api members: {len(members)} keys matched "
        f"({len(routes)} routes indexed; RequireRegion keys={len(region_keys)})\n"
    )


def main(argv: list[str] | None = None) -> int:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--root", type=Path, default=None, help="monorepo root")
    args = parser.parse_args(argv)
    try:
        root = args.root.resolve() if args.root else monorepo_root()
    except FileNotFoundError as e:
        print(f"ERROR: {e}", file=sys.stderr)
        return 2
    code, report = run_check(root)
    sys.stdout.write(report)
    return code


if __name__ == "__main__":
    raise SystemExit(main())
