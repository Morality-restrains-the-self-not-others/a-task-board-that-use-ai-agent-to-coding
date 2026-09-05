#!/usr/bin/env python3
"""对照 Go 服务 mountRoutes 与 openapi.yaml paths，防止文档漂移（多服务）。

配置：db/scripts/ci/go_openapi_services.yaml

若配置 openapi_internal：skip_mounted_prefixes 下的挂载改由内部规范覆盖，
不混入浏览器门户用的公开 openapi.yaml。

Exit codes:
  0 — 通过（无硬漂移；若仅 --warn-extra 则允许 stale warn）
  1 — 硬漂移（挂载缺文档，或 --fail-on-extra 且存在 OpenAPI 残留）
  2 — 配置 / IO 错误
"""

from __future__ import annotations

import argparse
import re
import sys
from dataclasses import dataclass, field
from pathlib import Path
from typing import Any

try:
    import yaml
except ImportError:  # pragma: no cover
    print("ERROR: PyYAML required (import yaml)", file=sys.stderr)
    raise SystemExit(2)

HTTP_METHODS = frozenset({"GET", "POST", "PUT", "PATCH", "DELETE", "HEAD", "OPTIONS"})


def monorepo_root() -> Path:
    here = Path(__file__).resolve()
    for parent in here.parents:
        if (parent / "db" / "registry.yaml").is_file():
            return parent
    raise FileNotFoundError("db/registry.yaml not found above checker")


ROOT = monorepo_root()
DEFAULT_CONFIG = ROOT / "db" / "scripts" / "ci" / "go_openapi_services.yaml"


@dataclass
class ServiceSpec:
    name: str
    handlers: Path
    openapi: Path
    openapi_internal: Path | None = None
    # complete: skip 前缀下挂载必须全部出现在 openapi_internal（taskAuth）
    # partial: 仅校验 internal 文件内路径与挂载一致；未文档化的 skip 挂载允许存在（Cloud）
    internal_coverage: str = "complete"
    optional: bool = False
    prefix_routers: list[str] = field(default_factory=list)
    skip_mounted_prefixes: list[str] = field(default_factory=list)
    ignore_mounted: list[str] = field(default_factory=list)
    # openapi 中声明的动态子路径（Contains+HasSuffix 组合路由无法由静态解析建模）
    ignore_openapi: list[str] = field(default_factory=list)


def _norm(path: str) -> str:
    p = path.strip()
    if not p.startswith("/"):
        p = "/" + p
    if not p.endswith("/") and "{" not in p:
        p += "/"
    return p


def _strip_method(lit: str) -> str:
    parts = lit.split(" ", 1)
    if len(parts) == 2 and parts[0] in HTTP_METHODS:
        return parts[1]
    return lit


def load_config(path: Path) -> tuple[list[str], dict[str, ServiceSpec]]:
    raw = yaml.safe_load(path.read_text(encoding="utf-8")) or {}
    default_ignore = [_norm(x) for x in (raw.get("default_ignore_mounted") or [])]
    services: dict[str, ServiceSpec] = {}
    for name, cfg in (raw.get("services") or {}).items():
        if not isinstance(cfg, dict):
            raise ValueError(f"service {name} must be a mapping")
        ignore = list(default_ignore)
        ignore.extend(_norm(x) for x in (cfg.get("ignore_mounted") or []))
        internal_raw = cfg.get("openapi_internal")
        coverage = str(cfg.get("internal_coverage") or "complete").strip().lower()
        if coverage not in ("complete", "partial"):
            raise ValueError(f"service {name}: internal_coverage must be complete|partial")
        services[name] = ServiceSpec(
            name=name,
            handlers=ROOT / cfg["handlers"],
            openapi=ROOT / cfg["openapi"],
            openapi_internal=(ROOT / internal_raw) if internal_raw else None,
            internal_coverage=coverage,
            optional=bool(cfg.get("optional")),
            prefix_routers=[_norm(x) for x in (cfg.get("prefix_routers") or [])],
            skip_mounted_prefixes=[_norm(x) for x in (cfg.get("skip_mounted_prefixes") or [])],
            ignore_mounted=ignore,
            ignore_openapi=[_norm(x) for x in (cfg.get("ignore_openapi") or [])],
        )
    return default_ignore, services


def extract_mount_routes(src: str, *, ignore_mounted: list[str]) -> set[str]:
    m = re.search(r"func mountRoutes\(.*?\)\s*\{", src)
    if not m:
        raise ValueError("mountRoutes not found")
    start = m.end()
    depth = 1
    i = start
    while i < len(src) and depth:
        if src[i] == "{":
            depth += 1
        elif src[i] == "}":
            depth -= 1
        i += 1
    body = src[start : i - 1]

    routes: set[str] = set()
    for lit in re.findall(r'(?:mux\.)?HandleFunc\("([^"]+)"', body):
        p = _strip_method(lit)
        if p == "/":
            continue
        routes.add(_norm(p))

    for suf in re.findall(r'HasSuffix\(\s*p\s*,\s*"([^"]+)"\s*\)', body):
        if suf.startswith("/billing/"):
            routes.add(_norm(f"/api/tenant/{{tenant_id}}{suf}"))
        else:
            routes.add(_norm(suf))

    for pref in re.findall(r'HasPrefix\(\s*p\s*,\s*"([^"]+)"\s*\)', body):
        routes.add(_norm(pref))

    ignore = {_norm(x) for x in ignore_mounted}
    return {r for r in routes if _norm(r) not in ignore}


def extract_openapi_paths(doc: dict[str, Any]) -> set[str]:
    return {_norm(str(p)) for p in (doc.get("paths") or {})}


def _skipped(mounted: str, prefixes: list[str]) -> bool:
    m = _norm(mounted)
    for pref in prefixes:
        pn = _norm(pref)
        if m == pn or m.startswith(pn.rstrip("/") + "/"):
            return True
    return False


def openapi_covers(mounted: str, openapi: set[str], prefix_routers: list[str]) -> bool:
    m = _norm(mounted)
    if m in openapi:
        return True
    alt = m.rstrip("/") if m.endswith("/") else m + "/"
    if alt in openapi:
        return True
    # Prefix router: OK if documented children exist OR router is declared
    if m in {_norm(x) for x in prefix_routers}:
        if any(op.startswith(m.rstrip("/") + "/") for op in openapi):
            return True
        # declared prefix router without children still OK (workspace batch etc.)
        return True
    for op in openapi:
        if op.startswith(m.rstrip("/") + "/{") or op.startswith(m):
            return True
    # segment match
    return any(_paths_seg_equal(m, op) for op in openapi)


def _segs(path: str) -> list[str]:
    return [s for s in path.strip("/").split("/") if s]


def _seg_match(a: str, b: str) -> bool:
    if (a.startswith("{") and a.endswith("}")) or (b.startswith("{") and b.endswith("}")):
        return True
    return a == b


def _paths_seg_equal(a: str, b: str) -> bool:
    sa, sb = _segs(a), _segs(b)
    if len(sa) != len(sb):
        return False
    return all(_seg_match(x, y) for x, y in zip(sa, sb))


def mounted_covers(openapi_path: str, mounted: set[str], prefix_routers: list[str]) -> bool:
    op = _norm(openapi_path)
    if op in mounted:
        return True
    alt = op.rstrip("/") if op.endswith("/") else op + "/"
    if alt in mounted:
        return True
    for pref in prefix_routers:
        pn = _norm(pref)
        if pn in mounted and (op == pn or op.startswith(pn.rstrip("/") + "/")):
            return True
    for m in mounted:
        if _paths_seg_equal(op, m):
            return True
    return False


def _check_pair(
    *,
    label: str,
    mounted: set[str],
    openapi: set[str],
    prefix_routers: list[str],
    ignore_openapi: list[str],
    warn_extra: bool,
    fail_on_extra: bool,
) -> tuple[int, list[str], int, int]:
    """Returns exit_code, report_lines, missing_count, stale_count."""
    missing = sorted(r for r in mounted if not openapi_covers(r, openapi, prefix_routers))
    stale = sorted(
        p
        for p in openapi
        if not mounted_covers(p, mounted, prefix_routers) and _norm(p) not in ignore_openapi
    )
    lines: list[str] = []
    exit_code = 0
    if missing:
        lines.append(f"DRIFT [{label}]: mounted routes missing from OpenAPI:")
        for r in missing:
            lines.append(f"  - {r}")
        exit_code = 1
    if stale and (warn_extra or fail_on_extra):
        tag = "DRIFT" if fail_on_extra else "WARN"
        lines.append(f"{tag} [{label}]: OpenAPI paths with no matching mountRoutes:")
        for r in stale:
            lines.append(f"  - {r}")
        if fail_on_extra:
            exit_code = 1
    return exit_code, lines, len(missing), len(stale)


def run_check_service(
    spec: ServiceSpec,
    handlers_src: str,
    openapi_doc: dict[str, Any],
    *,
    warn_extra: bool,
    fail_on_extra: bool,
    openapi_internal_doc: dict[str, Any] | None = None,
) -> tuple[int, str]:
    mounted = extract_mount_routes(handlers_src, ignore_mounted=spec.ignore_mounted)
    openapi = extract_openapi_paths(openapi_doc)

    public_mounted = {r for r in mounted if not _skipped(r, spec.skip_mounted_prefixes)}
    skipped_mounted = {r for r in mounted if _skipped(r, spec.skip_mounted_prefixes)}
    # 公开规范中的「非 skip」路径；skip 前缀下的路径可暂留在公开文件（Cloud）或迁入 internal
    public_openapi = {p for p in openapi if not _skipped(p, spec.skip_mounted_prefixes)}
    skip_prefixed_in_public = {p for p in openapi if _skipped(p, spec.skip_mounted_prefixes)}

    lines: list[str] = []
    exit_code = 0

    code, part, missing_n, stale_n = _check_pair(
        label=f"{spec.name}/public",
        mounted=public_mounted,
        openapi=public_openapi,
        prefix_routers=spec.prefix_routers,
        ignore_openapi=spec.ignore_openapi,
        warn_extra=warn_extra,
        fail_on_extra=fail_on_extra,
    )
    lines.extend(part)
    exit_code = max(exit_code, code)

    internal_paths = 0
    internal_missing = 0
    internal_stale = 0
    if spec.openapi_internal is not None:
        if openapi_internal_doc is None:
            raise ValueError("openapi_internal_doc required when openapi_internal is set")
        if skip_prefixed_in_public:
            lines.append(
                f"DRIFT [{spec.name}/public]: skip-prefix paths must live in "
                f"openapi-internal.yaml, not public openapi.yaml:"
            )
            for r in sorted(skip_prefixed_in_public):
                lines.append(f"  - {r}")
            exit_code = 1
        internal = extract_openapi_paths(openapi_internal_doc)
        internal_paths = len(internal)
        if spec.internal_coverage == "complete":
            mounts_for_internal = skipped_mounted
        else:
            # partial: 不要求全部 skip 挂载入文档；仅对「已写入 internal 的路径」做 stale
            mounts_for_internal = set()
        code_i, part_i, internal_missing, _ = _check_pair(
            label=f"{spec.name}/internal",
            mounted=mounts_for_internal,
            openapi=internal if spec.internal_coverage == "complete" else set(),
            prefix_routers=spec.prefix_routers,
            ignore_openapi=spec.ignore_openapi,
            warn_extra=False,
            fail_on_extra=False,
        )
        # partial 模式下 _check_pair 的 missing 恒为空；complete 用上面结果
        if spec.internal_coverage == "partial":
            part_i = []
            internal_missing = 0
            code_i = 0
        stale_internal = sorted(
            p for p in internal if not mounted_covers(p, mounted, spec.prefix_routers)
        )
        internal_stale = len(stale_internal)
        lines.extend(part_i)
        if code_i:
            exit_code = max(exit_code, code_i)
        if stale_internal and (warn_extra or fail_on_extra):
            tag = "DRIFT" if fail_on_extra else "WARN"
            lines.append(
                f"{tag} [{spec.name}/internal]: OpenAPI paths with no matching mountRoutes:"
            )
            for r in stale_internal:
                lines.append(f"  - {r}")
            if fail_on_extra:
                exit_code = 1
    elif skip_prefixed_in_public:
        # 公开文件里文档化的 skip 前缀路径仍须对应真实挂载（防 stale）
        stale_skip = sorted(
            p
            for p in skip_prefixed_in_public
            if not mounted_covers(p, mounted, spec.prefix_routers)
        )
        if stale_skip and (warn_extra or fail_on_extra):
            tag = "DRIFT" if fail_on_extra else "WARN"
            lines.append(
                f"{tag} [{spec.name}/public]: skip-prefix OpenAPI paths with no matching mount:"
            )
            for r in stale_skip:
                lines.append(f"  - {r}")
            if fail_on_extra:
                exit_code = 1
            stale_n += len(stale_skip)

    if exit_code == 0:
        suffix = f", {stale_n} public stale-doc warnings" if stale_n and warn_extra else ""
        internal_note = ""
        if spec.openapi_internal is not None:
            internal_note = (
                f"; internal={internal_paths} paths "
                f"(skip-mounted={len(skipped_mounted)})"
            )
        lines.append(
            f"OK [{spec.name}]: openapi covers mountRoutes "
            f"({len(mounted)} mounted, {len(public_openapi)} public paths"
            f"{internal_note}{suffix})"
        )
    else:
        lines.append(
            f"[{spec.name}] mounted={len(mounted)} public_openapi={len(public_openapi)} "
            f"missing={missing_n} stale={stale_n}"
            + (
                f" internal_missing={internal_missing} internal_stale={internal_stale}"
                if spec.openapi_internal is not None
                else ""
            )
        )
    return exit_code, "\n".join(lines) + "\n"


def main(argv: list[str] | None = None) -> int:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--config", type=Path, default=DEFAULT_CONFIG)
    parser.add_argument(
        "--service",
        action="append",
        dest="services",
        help="Service name (repeatable). Default: all in config",
    )
    parser.add_argument(
        "--warn-extra",
        action=argparse.BooleanOptionalAction,
        default=True,
        help="Warn on OpenAPI-only paths (default: on)",
    )
    parser.add_argument(
        "--fail-on-extra",
        action="store_true",
        help="Fail when OpenAPI documents paths not mounted in code",
    )
    args = parser.parse_args(argv)

    try:
        _, services = load_config(args.config)
    except Exception as exc:  # noqa: BLE001
        print(f"ERROR: config: {exc}", file=sys.stderr)
        return 2

    selected = args.services or sorted(services.keys())
    exit_code = 0
    reports: list[str] = []
    for name in selected:
        spec = services.get(name)
        if not spec:
            print(f"ERROR: unknown service {name!r}; known={sorted(services)}", file=sys.stderr)
            return 2
        if not spec.openapi.is_file():
            if spec.optional:
                reports.append(f"SKIP [{name}]: openapi.yaml missing (optional=true)")
                continue
            print(f"ERROR [{name}]: openapi not found: {spec.openapi}", file=sys.stderr)
            return 2
        if not spec.handlers.is_file():
            if spec.optional:
                reports.append(f"SKIP [{name}]: handlers missing (optional=true)")
                continue
            print(f"ERROR [{name}]: handlers not found: {spec.handlers}", file=sys.stderr)
            return 2
        try:
            handlers_src = spec.handlers.read_text(encoding="utf-8")
            doc = yaml.safe_load(spec.openapi.read_text(encoding="utf-8")) or {}
            internal_doc = None
            if spec.openapi_internal is not None:
                if not spec.openapi_internal.is_file():
                    print(
                        f"ERROR [{name}]: openapi_internal not found: {spec.openapi_internal}",
                        file=sys.stderr,
                    )
                    return 2
                internal_doc = (
                    yaml.safe_load(spec.openapi_internal.read_text(encoding="utf-8")) or {}
                )
            code, report = run_check_service(
                spec,
                handlers_src,
                doc,
                warn_extra=args.warn_extra,
                fail_on_extra=args.fail_on_extra,
                openapi_internal_doc=internal_doc,
            )
        except OSError as exc:
            print(f"ERROR [{name}]: {exc}", file=sys.stderr)
            return 2
        except Exception as exc:  # noqa: BLE001
            print(f"ERROR [{name}] parse: {exc}", file=sys.stderr)
            return 2
        reports.append(report.rstrip("\n"))
        if code != 0:
            exit_code = code

    sys.stdout.write("\n".join(reports) + "\n")
    return exit_code


if __name__ == "__main__":
    raise SystemExit(main())
