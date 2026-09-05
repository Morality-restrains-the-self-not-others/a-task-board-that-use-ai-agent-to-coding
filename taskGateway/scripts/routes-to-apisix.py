#!/usr/bin/env python3
"""Generate APISIX standalone apisix.yaml from routes/routes.yaml + conf/gateway/task-gateway (merged with conf-local)."""

from __future__ import annotations

import argparse
import html
import os
import re
import sys
from pathlib import Path
from urllib.parse import urlparse

import yaml

from cors_allow_headers import cors_allow_headers_apisix, cors_allow_headers_go, cors_expose_headers_apisix, cors_expose_headers_go, CORS_ALLOW_HEADERS

ROOT = Path(__file__).resolve().parents[1]
REPO = ROOT.parent
ROUTES_SRC = ROOT / "routes" / "routes.yaml"
DOCKER_INFRA_CONF = REPO / "conf" / "infra" / "docker-infra" / "config.yaml"
OUT = ROOT / "apisix" / "apisix.yaml"
PORTAL_OUT = ROOT / "generated" / "docs-portal" / "index.html"
OPS_PORTAL_OUT = ROOT / "generated" / "docs-portal" / "ops" / "index.html"
GATEWAYCORS_OUT = REPO / "shareLib" / "gatewaycors" / "allow_headers_gen.go"

# Add runAll/scripts to path for domain service imports
_RUNALL_SCRIPTS = REPO / "runAll" / "scripts"
if str(_RUNALL_SCRIPTS) not in sys.path:
    sys.path.insert(0, str(_RUNALL_SCRIPTS))

from conf_local import merge_conf_local, overlay_conf_file  # noqa: E402

_GATEWAY_CONF_REL = "gateway/task-gateway/config.yaml"


def _read_tracked_gateway_conf(root: Path) -> dict:
    conf_path = root / "conf" / _GATEWAY_CONF_REL
    conf = yaml.safe_load(conf_path.read_text(encoding="utf-8")) if conf_path.is_file() else {}
    if not isinstance(conf, dict):
        return {}
    return conf


def load_gateway_conf(repo: Path | None = None) -> dict:
    """Load gateway conf and overlay conf-local (ADR-0054), walking parents.

    Tracked conf/gateway/task-gateway/config.yaml keeps gatewayInternalSecret
    empty. clone-run writes the real value to $DEPLOY_ROOT/conf-local, while
    this script's REPO is often envs/current (no conf-local). Walking parents
    finds the overlay; stopping at the first non-empty secret.

    OPT-20260901-006: when run.sh routes_apply sources cutover.env it exports
    DEPLOY_ROOT/CONF_ROOT explicitly. Prefer that root over the upward walk so
    a stray non-empty conf-local in an intermediate dir (e.g. envs/current)
    can't short-circuit resolution before reaching the deploy root.
    """
    start = repo if repo is not None else REPO
    conf = _read_tracked_gateway_conf(start)
    here = start.resolve()
    deploy_root = os.environ.get("DEPLOY_ROOT", "").strip()
    conf_root = os.environ.get("CONF_ROOT", "").strip()
    if deploy_root:
        here = Path(deploy_root).resolve()
    elif conf_root:
        cr = Path(conf_root).resolve()
        here = cr.parent if cr.name == "conf" else cr
    for _ in range(8):
        conf = merge_conf_local(here, _GATEWAY_CONF_REL, conf)
        if str(conf.get("gatewayInternalSecret") or "").strip():
            return conf
        parent = here.parent
        if parent == here:
            break
        here = parent
    return conf


def _load() -> tuple[dict, dict]:
    routes_doc = yaml.safe_load(ROUTES_SRC.read_text(encoding="utf-8"))
    return routes_doc, load_gateway_conf(REPO)


def _docs_enabled(conf: dict) -> bool:
    env = os.environ.get("TASK_GATEWAY_DOCS_ENABLED", "").strip().lower()
    if env in ("1", "true", "yes"):
        return True
    if env in ("0", "false", "no"):
        return False
    return bool((conf.get("docs") or {}).get("enabled", False))


def _resolve_auth_mode(rule: dict) -> str:
    """Resolve the canonical auth_mode from a route rule.

    Supported modes:
      - deny   → block with fault-injection 403
      - token  → forward-auth via taskAuth + proxy-rewrite gateway secret
      - public → no gateway auth (truly public: login, health, OIDC, SPA)
      - machine→ no gateway auth (service-level auth: webhooks, container tokens)
      - none   → deprecated alias for 'public' (backward compatible)
      - app-jwt→ deprecated; routes should migrate to 'token' (P0 cleanup)
    """
    if "auth_mode" in rule:
        mode = str(rule["auth_mode"]).strip()
        # Canonicalise deprecated aliases
        if mode == "none":
            return "public"
        if mode == "app-jwt":
            # app-jwt was removed (no issuer exists post-Django decommission).
            # Any remaining routes must be migrated to 'token'.
            raise SystemExit(
                f"ERROR: route {rule.get('id', '?')}: auth_mode 'app-jwt' is removed. "
                f"Migrate to 'token' or use 'machine' for service-level auth."
            )
        return mode
    legacy = rule.get("auth")
    if legacy == "deny":
        return "deny"
    if legacy == "public":
        return "public"
    return "token"


# Map retained for documentation / future public-host lookups (upstream hosts
# use docker.upstreamHost / loopback — see _upstream_host).
_UPSTREAM_KEY_MAP = {
    "django": "api",
    "taskAuth": "auth",
    "gitOauth": "gitoauth",
}


_cached_scheme = None
_cached_infra_host: str | None = None


def _get_addressing_scheme():
    """Load AddressingScheme from base.yaml once per process (cached)."""
    global _cached_scheme
    if _cached_scheme is None:
        from domain.services.base_yaml_loader import BaseYAMLLoader
        loader = BaseYAMLLoader()
        base_path = str(REPO / "conf" / "base.yaml")
        _cached_scheme = loader.load(base_path)
    return _cached_scheme


def _resolve_infra_host() -> str:
    """Resolve ${INFRA_HOST}: env → docker-infra.infraHost → localhost.

    Resolution order:
    1. INFRA_HOST env var (highest priority)
    2. conf/infra/docker-infra/config.yaml → infraHost (with ${INFRA_HOST:-default} expansion)
    3. localhost (fallback)
    """
    global _cached_infra_host
    if _cached_infra_host is not None:
        return _cached_infra_host
    env = os.environ.get("INFRA_HOST", "").strip()
    if env:
        _cached_infra_host = env
        return _cached_infra_host
    if DOCKER_INFRA_CONF.is_file():
        doc = overlay_conf_file(DOCKER_INFRA_CONF)
        host = str(doc.get("infraHost") or "").strip()
        if host:
            # The infraHost value may itself contain ${INFRA_HOST:-default} —
            # expand it so we get the actual IP, not a template literal.
            # At this point INFRA_HOST env var is known to be unset, so the
            # expansion will use the default value from the template.
            host = re.sub(r"\$\{INFRA_HOST:-([^}]*)\}", r"\1", host)
            if host:
                _cached_infra_host = host
                return _cached_infra_host
    _cached_infra_host = "localhost"
    return _cached_infra_host


def _expand_infra_host_template(value: str) -> str:
    """Replace ${INFRA_HOST:-default} then ${INFRA_HOST} in CORS origin strings."""
    host = _resolve_infra_host()
    # Expand :-default forms first so bare ${INFRA_HOST} replace does not corrupt them.
    out = re.sub(r"\$\{INFRA_HOST:-([^}]*)\}", host, value)
    return out.replace("${INFRA_HOST}", host)


def _upstream_host(conf: dict, upstreams: dict, name: str, logical_host: str) -> str:
    """Resolve APISIX upstream host (loopback / docker bridge — never public DNS).

    Public domain addresses in conf/base.yaml are for browsers, CORS, and OIDC.
    Upstream nodes must reach host-published service ports locally.
    """
    docker = conf.get("docker") or {}
    in_docker = os.environ.get("TASK_GATEWAY_APISIX_IN_DOCKER", "").strip() in ("1", "true", "yes")
    if in_docker and name == "docsPortal":
        return str((conf.get("docs") or {}).get("portalUpstream", {}).get("host") or "docs-portal")
    if name == "docsPortal":
        docs = conf.get("docs") or {}
        portal = docs.get("portalUpstream") or {}
        return str(portal.get("nativeHost") or portal.get("host") or "127.0.0.1")

    if in_docker:
        upstream_host = docker.get("upstreamHost")
        if upstream_host:
            return str(upstream_host).strip()
        raise ValueError(
            "docker.upstreamHost must be configured in conf/gateway/task-gateway/config.yaml "
            "when TASK_GATEWAY_APISIX_IN_DOCKER=1"
        )
    return logical_host or "127.0.0.1"


def _upstream_port(upstream_def: dict, conf: dict, name: str) -> int:
    if name == "docsPortal":
        docs = conf.get("docs") or {}
        portal = docs.get("portalUpstream") or {}
        if os.environ.get("TASK_GATEWAY_APISIX_IN_DOCKER", "").strip() in ("1", "true", "yes"):
            return int(portal.get("port") or 80)
        return int(portal.get("nativePort") or portal.get("port") or 9088)
    return int(upstream_def.get("port") or 80)


def _upstream_nodes(upstreams: dict, name: str, conf: dict, upstream_def: dict | None = None) -> dict:
    if upstream_def is None:
        upstream_def = upstreams.get(name) or {}
    host = _upstream_host(conf, upstreams, name, str(upstream_def.get("host") or "127.0.0.1"))
    port = _upstream_port(upstream_def, conf, name)
    return {f"{host}:{port}": 1}


def _upstream_health_checks(upstream_def: dict | None) -> dict:
    """Emit APISIX active health-check config from an optional `healthCheck` key.

    Multi-replica / future scale-out: APISIX periodically probes the upstream and
    removes unhealthy nodes from round-robin, so transient Connection-refused
    502s surface only while a node is marked down (single-node dev box gains
    observability value, not failover). See OPT-20260811-008.
    """
    if not isinstance(upstream_def, dict):
        return {}
    hc = upstream_def.get("healthCheck")
    if not isinstance(hc, dict):
        return {}
    http_path = str(hc.get("http_path") or "/api/health/").strip()
    if not http_path.startswith("/"):
        raise ValueError(f"healthCheck.http_path must start with /: {http_path!r}")
    active = {
        "type": "http",
        "http_path": http_path,
        "timeout": int(hc.get("timeout") or 3),
        "healthy": {
            "interval": int(hc.get("healthy_interval") or 5),
            "successes": int(hc.get("successes") or 2),
        },
        "unhealthy": {
            "interval": int(hc.get("unhealthy_interval") or 5),
            "http_failures": int(hc.get("http_failures") or 3),
        },
    }
    # host header is optional; APISIX schema rejects the empty string, so only
    # emit it when an explicit probe Host header is configured.
    host = str(hc.get("host") or "").strip()
    if host:
        active["host"] = host
    return {"checks": {"active": active}}


def _upstream_retries(upstream_def: dict | None) -> dict:
    """Emit APISIX upstream retry config from an optional `retry` key.

    `retry.retries` (integer; APISIX 默认 1) 与 `retry.retry_timeout`（秒 —
    APISIX balancer 将其与 ngx.now() 相加作为重试截止，见 balancer.lua
    `ctx.proxy_retry_deadline = ngx_now() + up_conf.retry_timeout`）。
    精准编译重启无监听间隙内 connection-refused 由 APISIX 在 retry_timeout
    窗口内继续重试，吸收秒级 502（OPT-20260827-039）。未配置 `retry` 时不
    发射任何字段，保持既有 upstream 形状（防止全站行为变化）。
    """
    if not isinstance(upstream_def, dict):
        return {}
    r = upstream_def.get("retry")
    if not isinstance(r, dict):
        return {}
    out = {}
    retries = r.get("retries")
    if retries is not None:
        out["retries"] = int(retries)
    retry_timeout = r.get("retry_timeout")
    if retry_timeout is not None:
        out["retry_timeout"] = int(retry_timeout)
    return out


def _parse_docs_paths(docs: dict, *, field_name: str) -> dict:
    ui = str(docs.get("uiPath") or "").strip()
    schema = str(docs.get("schemaPath") or "").strip()
    if not ui.startswith("/") or not schema.startswith("/"):
        raise ValueError(f"{field_name} paths must start with /: {docs}")
    return {"uiPath": ui, "schemaPath": schema}


def _parse_upstream_docs(upstream_def: dict) -> dict | None:
    docs = upstream_def.get("docs")
    if docs is False or docs is None:
        return None
    if isinstance(docs, dict):
        return _parse_docs_paths(docs, field_name="upstream docs")
    raise ValueError(f"upstream docs must be object or false, got: {docs!r}")


def _parse_upstream_docs_internal(upstream_def: dict) -> dict | None:
    """Optional ops-only OpenAPI (never listed on public /gateway/docs/)."""
    docs = upstream_def.get("docsInternal")
    if docs is False or docs is None:
        return None
    if isinstance(docs, dict):
        return _parse_docs_paths(docs, field_name="upstream docsInternal")
    raise ValueError(f"upstream docsInternal must be object or false, got: {docs!r}")


def validate_upstream_docs_contract(upstream_defs: dict) -> None:
    for name, upstream_def in upstream_defs.items():
        if not isinstance(upstream_def, dict):
            raise ValueError(f"upstream {name} must be a mapping")
        if "docs" not in upstream_def:
            raise ValueError(f"upstream {name} missing docs (object or false)")
        # docsInternal is optional; parse to validate shape when present
        _parse_upstream_docs_internal(upstream_def)


def _forward_auth_plugin(conf: dict, upstreams: dict) -> dict:
    up = upstreams.get("taskAuth") or {}
    host = _upstream_host(
        conf, upstreams, "taskAuth", str(up.get("host") or conf.get("host") or "127.0.0.1")
    )
    port = int(up.get("port") or 8003)
    path = str((conf.get("auth") or {}).get("forwardAuthPath") or "/api/internal/gateway/forward-auth/")
    secret = str(
        (conf.get("auth") or {}).get("taskauthInternalSecret")
        or conf.get("gatewayInternalSecret")
        or ""
    )
    return {
        "forward-auth": {
            "uri": f"http://{host}:{port}{path}",
            # 透传全链路 trace 头：taskAuth tracelog 中间件要求 X-Trace-Id 必须
            # 伴随 X-Parent-Span-Id/traceparent，否则 400；且其 401/错误响应
            # body.trace_id 才能回显客户端 traceId（元规则 data-traceId）
            "request_headers": [
                "Authorization",
                "Cookie",
                "X-Trace-Id",
                "X-Parent-Span-Id",
                "traceparent",
            ],
            "upstream_headers": [
                "X-User-Id",
                # OPT-20260807-010: 主邮箱头必须透传 —— taskAuth writeForwardAuthHeaders
                # 注入 X-User-Email（resolveUserEmail 主邮箱，供下游邮箱匹配），白名单
                # 缺失会被 APISIX 静默丢弃，导致 taskAiProvider 厂商门户资格判断
                # （vendor-status has_email / vendor-application 邮箱前置校验）永远
                # 视为未绑定邮箱，用户已绑定仍报「厂商门户需先绑定邮箱账号」。
                "X-User-Email",
                "X-Gateway-Auth-Verified",
                "X-Auth-Tenant-Claims",
                # v63 RBAC：forward-auth 响应的平台角色与租户权限码必须透传给上游，
                # 否则 shareLib authz（RequirePlatformPerm/RequirePerm）永远拿不到权限头。
                # 遗留 X-Auth-Superuser/X-Auth-Staff 已移除（消费方已迁移 X-User-Roles）。
                "X-User-Roles",
                "X-User-Is-Tester",
                "X-Tenant-Perms",
                "X-Impersonator-Id",
                "X-Impersonation-Session-Id",
                "X-Impersonating",
            ],
            "headers": {"X-TaskAuth-Internal-Secret": secret},
        }
    }


def _token_route_transform(conf: dict) -> dict:
    gw_secret = str(conf.get("gatewayInternalSecret") or "")
    return {
        "proxy-rewrite": {
            "headers": {
                "set": {"X-TaskGateway-Internal-Secret": gw_secret},
            }
        }
    }


_TRACE_PROPAGATION_LUA = r"""
return function(conf, ctx)
    local core = require("apisix.core")
    local resty_random = require("resty.random")
    local str = require("resty.string")
    local trace_id = core.request.header(ctx, "X-Trace-Id")
    if not trace_id or trace_id == "" then
        -- Auto-generate traceId so error responses (404, 500, etc.)
        -- always carry a traceable ID for diagnostics.
        trace_id = str.to_hex(resty_random.bytes(16))
        core.request.set_header(ctx, "X-Trace-Id", trace_id)
    end
    -- Set response header so the client always receives X-Trace-Id,
    -- even for APISIX-generated error responses (404 route-not-found, etc.).
    core.response.set_header("X-Trace-Id", trace_id)
    if not core.request.header(ctx, "X-Parent-Span-Id") and not core.request.header(ctx, "traceparent") then
        local span = str.to_hex(resty_random.bytes(8))
        core.request.set_header(ctx, "X-Parent-Span-Id", span)
    end
end
"""


def _trace_plugin() -> dict:
    """Complete inbound trace propagation for upstream services.

    Auto-generates X-Trace-Id when the client omitted trace headers, sets it
    as both request and response header, and adds X-Parent-Span-Id for services
    that require strict trace middleware.  The response header ensures even
    APISIX-generated error responses (404, 500) carry a traceable ID.
    """
    return {
        "serverless-pre-function": {
            "phase": "rewrite",
            "functions": [_TRACE_PROPAGATION_LUA.strip()],
        },
    }


def _file_logger_plugin() -> dict:
    return {
        "file-logger": {
            "path": "/usr/local/apisix/logs/taskgateway-access.log",
            "include_req_body": False,
            "include_resp_body": False,
        }
    }


def _prometheus_plugin() -> dict:
    """Expose gateway-level HTTP latency/status/bandwidth via /apisix/prometheus/metrics.

    OPT-20260827-003: user-perceived latency includes APISIX routing / forward-auth /
    upstream queueing, not just the downstream Go service *_http_request_duration_seconds.
    prefer_name makes the route label carry the route name instead of numeric id.
    """
    return {
        "prometheus": {
            "prefer_name": True,
        }
    }


def _login_rate_limit(conf: dict) -> dict:
    rl = conf.get("rateLimit") or {}
    per_min = int(rl.get("loginPerMinute") or 30)
    return {
        "limit-req": {
            "rate": per_min / 60.0,
            "burst": max(5, per_min // 3),
            "rejected_code": 429,
            "key": "remote_addr",
        }
    }


def _global_rate_limit(conf: dict) -> dict:
    """Wire conf rateLimit.globalPerMinute into APISIX global_rules (was previously unused)."""
    rl = conf.get("rateLimit") or {}
    per_min = int(rl.get("globalPerMinute") or 0)
    if per_min <= 0:
        return {}
    return {
        "limit-req": {
            "rate": per_min / 60.0,
            "burst": max(20, per_min // 10),
            "rejected_code": 429,
            "key": "remote_addr",
        }
    }


def _cors_plugin(conf: dict) -> dict:
    cors = conf.get("cors") or {}
    origins = cors.get("allowedOrigins") or ["*"]
    # Resolve ${subdomains.xxx} / ${INFRA_HOST} template variables in CORS origins
    if isinstance(origins, list):
        scheme = _get_addressing_scheme()
        origins = [
            _expand_infra_host_template(scheme.resolve_template(o) if isinstance(o, str) else o)
            if isinstance(o, str)
            else o
            for o in origins
        ]
    return {
        "cors": {
            "allow_origins": ",".join(origins) if isinstance(origins, list) else str(origins),
            "allow_methods": "GET,POST,PUT,PATCH,DELETE,OPTIONS,HEAD",
            "allow_headers": cors_allow_headers_apisix(),
            "expose_headers": cors_expose_headers_apisix(),
            "allow_credential": bool(cors.get("allowCredentials", True)),
            "max_age": 3600,
        }
    }


def _deny_route(rule: dict) -> dict:
    return {
        "id": rule["id"],
        "priority": int(rule.get("priority", 0)),
        "uri": rule["uri"],
        "upstream_id": "up-deny",
        "plugins": {
            "fault-injection": {
                "abort": {
                    "http_status": 403,
                    "body": '{"detail":"forbidden"}',
                }
            }
        },
    }


def _build_route(rule: dict, upstreams: dict, conf: dict) -> dict:
    mode = _resolve_auth_mode(rule)
    if mode == "deny":
        return _deny_route(rule)

    upstream_name = rule["upstream"]
    upstream_id = f"up-{upstream_name}"
    plugins: dict = {}
    if mode == "token":
        plugins.update(_token_route_transform(conf))
        plugins.update(_forward_auth_plugin(conf, upstreams))
    plugins.update(_cors_plugin(conf))
    plugins.update(_trace_plugin())
    if rule.get("id") == "taskauth-login":
        plugins.update(_login_rate_limit(conf))

    entry: dict = {
        "id": rule["id"],
        "priority": int(rule.get("priority", 0)),
        "upstream_id": upstream_id,
        "plugins": plugins,
    }
    if "uris" in rule:
        entry["uris"] = list(rule["uris"])
    else:
        entry["uri"] = rule["uri"]
    methods = rule.get("methods")
    if methods:
        entry["methods"] = list(methods)
    # Resolve hosts template variables (${subdomains.base}, ${subdomains.www}, …)
    if "hosts" in rule:
        scheme = _get_addressing_scheme()
        entry["hosts"] = [scheme.resolve_template(h) for h in rule["hosts"]]
    return entry


def _proxy_rewrite_strip_prefix(prefix: str) -> dict:
    if not prefix.endswith("/"):
        prefix = prefix + "/"
    escaped = prefix.replace("/", r"\/")
    return {
        "proxy-rewrite": {
            "regex_uri": [f"^{escaped}(.*)", "/$1"],
        }
    }


def _proxy_rewrite_strip_gateway_prefix(upstream_name: str) -> dict:
    return _proxy_rewrite_strip_prefix(f"/gateway/docs/{upstream_name}/")


def _proxy_rewrite_strip_ops_prefix(upstream_name: str) -> dict:
    return _proxy_rewrite_strip_prefix(f"/gateway/ops/docs/{upstream_name}/")


def _proxy_rewrite_schema(schema_path: str) -> dict:
    parsed = urlparse(schema_path)
    uri = parsed.path or "/"
    if parsed.query:
        uri = f"{uri}?{parsed.query}"
    return {"proxy-rewrite": {"uri": uri}}


def _docs_base_plugins(conf: dict) -> dict:
    return {**_cors_plugin(conf), **_trace_plugin()}


def _ops_auth_plugins(conf: dict, upstreams: dict) -> dict:
    """Token + forward-auth — ops OpenAPI must not be anonymously browsable."""
    return {
        **_token_route_transform(conf),
        **_forward_auth_plugin(conf, upstreams),
        **_docs_base_plugins(conf),
    }


def _build_docs_routes(upstream_defs: dict, conf: dict) -> list[dict]:
    routes: list[dict] = []
    base_plugins = lambda: _docs_base_plugins(conf)

    routes.append(
        {
            "id": "gateway-docs-portal",
            "priority": 2101,
            "uris": ["/gateway/docs/", "/gateway/docs"],
            "upstream_id": "up-docsPortal",
            "plugins": base_plugins(),
        }
    )

    for name, upstream_def in upstream_defs.items():
        docs = _parse_upstream_docs(upstream_def)
        if not docs:
            continue
        routes.append(
            {
                "id": f"gateway-docs-{name}-ui",
                "priority": 2100,
                "uri": f"/gateway/docs/{name}/*",
                "upstream_id": f"up-{name}",
                "plugins": {
                    **_proxy_rewrite_strip_gateway_prefix(name),
                    **base_plugins(),
                },
            }
        )
        routes.append(
            {
                "id": f"gateway-openapi-{name}-schema",
                "priority": 2100,
                "uri": f"/gateway/openapi/{name}.json",
                "upstream_id": f"up-{name}",
                "plugins": {
                    **_proxy_rewrite_schema(docs["schemaPath"]),
                    **base_plugins(),
                },
            }
        )

    # Ops-only internal OpenAPI portal (auth required; not mixed into public docs list)
    ops_plugins = _ops_auth_plugins(conf, upstream_defs)
    routes.append(
        {
            "id": "gateway-ops-docs-portal",
            "priority": 2103,
            "uris": ["/gateway/ops/docs/", "/gateway/ops/docs"],
            "upstream_id": "up-docsPortal",
            "plugins": {
                **_proxy_rewrite_schema("/ops/index.html"),
                **ops_plugins,
            },
        }
    )
    for name, upstream_def in upstream_defs.items():
        docs_int = _parse_upstream_docs_internal(upstream_def)
        if not docs_int:
            continue
        routes.append(
            {
                "id": f"gateway-ops-docs-{name}-ui",
                "priority": 2102,
                "uri": f"/gateway/ops/docs/{name}/*",
                "upstream_id": f"up-{name}",
                "plugins": {
                    **_proxy_rewrite_strip_ops_prefix(name),
                    **ops_plugins,
                },
            }
        )
        routes.append(
            {
                "id": f"gateway-ops-openapi-{name}-schema",
                "priority": 2102,
                "uri": f"/gateway/ops/openapi/{name}.json",
                "upstream_id": f"up-{name}",
                "plugins": {
                    **_proxy_rewrite_schema(docs_int["schemaPath"]),
                    **ops_plugins,
                },
            }
        )
    return routes


def _write_portal_html(upstream_defs: dict, conf: dict) -> None:
    public_base = str(conf.get("publicBase") or "https://localhost:8443").rstrip("/")
    items: list[str] = []
    for name, upstream_def in sorted(upstream_defs.items()):
        docs = _parse_upstream_docs(upstream_def)
        if docs:
            # Append uiPath so /gateway/docs/{name}/api/swagger/ rewrites to /api/swagger/
            ui_path = str(docs.get("uiPath") or "/").strip() or "/"
            if not ui_path.startswith("/"):
                ui_path = "/" + ui_path
            ui_url = f"{public_base}/gateway/docs/{name}{ui_path}"
            schema_url = f"{public_base}/gateway/openapi/{name}.json"
            items.append(
                f"<li><strong>{html.escape(name)}</strong> — "
                f'<a href="{html.escape(ui_url)}">Swagger UI</a> · '
                f'<a href="{html.escape(schema_url)}">OpenAPI JSON</a></li>'
            )
        else:
            items.append(
                f"<li><strong>{html.escape(name)}</strong> — "
                f"<em>未提供 OpenAPI（docs: false）</em></li>"
            )
    body = "\n".join(items)
    page = f"""<!DOCTYPE html>
<html lang="zh-CN">
<head>
  <meta charset="utf-8"/>
  <title>taskGateway API Docs</title>
  <style>
    body {{ font-family: system-ui, sans-serif; margin: 2rem; }}
    code {{ background: #f4f4f4; padding: 0.1rem 0.3rem; }}
  </style>
</head>
<body>
  <h1>taskGateway — 分服务 API 文档</h1>
  <p>网关入口：<code>{html.escape(public_base)}</code>。Try it out 请在 Swagger UI 中填写
  <code>Authorization: Token &lt;key&gt;</code>。</p>
  <ul>
{body}
  </ul>
</body>
</html>
"""
    PORTAL_OUT.parent.mkdir(parents=True, exist_ok=True)
    PORTAL_OUT.write_text(page, encoding="utf-8")


def _write_ops_portal_html(upstream_defs: dict, conf: dict) -> None:
    """Ops-only index for docsInternal; served at /gateway/ops/docs/ (token required)."""
    public_base = str(conf.get("publicBase") or "https://localhost:8443").rstrip("/")
    items: list[str] = []
    for name, upstream_def in sorted(upstream_defs.items()):
        docs_i = _parse_upstream_docs_internal(upstream_def)
        if not docs_i:
            continue
        ui_path = str(docs_i.get("uiPath") or "/").strip() or "/"
        if not ui_path.startswith("/"):
            ui_path = "/" + ui_path
        ui_url = f"{public_base}/gateway/ops/docs/{name}{ui_path}"
        schema_url = f"{public_base}/gateway/ops/openapi/{name}.json"
        items.append(
            f"<li><strong>{html.escape(name)}</strong> — "
            f'<a href="{html.escape(ui_url)}">Internal Swagger UI</a> · '
            f'<a href="{html.escape(schema_url)}">OpenAPI JSON</a></li>'
        )
    if not items:
        items.append("<li><em>暂无登记 docsInternal 的服务</em></li>")
    body = "\n".join(items)
    page = f"""<!DOCTYPE html>
<html lang="zh-CN">
<head>
  <meta charset="utf-8"/>
  <title>taskGateway Ops API Docs</title>
  <style>
    body {{ font-family: system-ui, sans-serif; margin: 2rem; }}
    code {{ background: #f4f4f4; padding: 0.1rem 0.3rem; }}
    .warn {{ color: #8a4b08; background: #fff8e6; padding: 0.75rem 1rem; border-radius: 4px; }}
  </style>
</head>
<body>
  <h1>taskGateway — 运维内部 API 文档</h1>
  <p class="warn">本页需有效 <code>Authorization: Token &lt;key&gt;</code>（forward-auth）。
  不在公开 <code>/gateway/docs/</code> 列出；请勿将链接嵌入面向终端用户的门户。</p>
  <p>网关入口：<code>{html.escape(public_base)}</code></p>
  <ul>
{body}
  </ul>
</body>
</html>
"""
    OPS_PORTAL_OUT.parent.mkdir(parents=True, exist_ok=True)
    OPS_PORTAL_OUT.write_text(page, encoding="utf-8")


def _render_gatewaycors_go() -> str:
    """Return expected allow_headers_gen.go contents from CORS SSOT."""
    names_literal = ",\n\t".join(f'"{h}"' for h in CORS_ALLOW_HEADERS)
    go_headers = cors_allow_headers_go().replace("\\", "\\\\").replace('"', '\\"')
    go_expose = cors_expose_headers_go().replace("\\", "\\\\").replace('"', '\\"')
    return f"""// Code generated by taskGateway/scripts/routes-to-apisix.py; DO NOT EDIT.

package gatewaycors

// AllowHeaders is the canonical CORS allow-headers list for taskGateway platform Go services.
const AllowHeaders = "{go_headers}"

// ExposeHeaders is the canonical Access-Control-Expose-Headers value for platform Go services.
// Exposing X-Trace-Id enables frontend data-traceId binding for error diagnostics.
const ExposeHeaders = "{go_expose}"

// CORSAllowHeaderNames is the ordered header name list (codegen SSOT).
var CORSAllowHeaderNames = []string{{
\t{names_literal},
}}
"""


def _write_gatewaycors_go() -> None:
    """Emit shareLib/gatewaycors allow_headers_gen.go from CORS SSOT."""
    GATEWAYCORS_OUT.parent.mkdir(parents=True, exist_ok=True)
    GATEWAYCORS_OUT.write_text(_render_gatewaycors_go(), encoding="utf-8")


def validate_installed_images_route_to_task_cloud(routes_doc: dict) -> None:
    """installed-images 必须路由到 taskCloudService（Django→Go 迁移约束）。"""
    required = {
        "/api/cloud/*",
        # installed-images is covered by /api/cloud/ convention dispatcher
        # (tenant_id passed via key=value path params)
    }
    for rule in routes_doc.get("routes") or []:
        if rule.get("id") != "task-cloud-service":
            continue
        uris = set(rule.get("uris") or [])
        missing = sorted(required - uris)
        if missing:
            raise ValueError(
                "task-cloud-service 缺少 installed-images 路由: " + ", ".join(missing)
            )
        return
    raise ValueError("routes.yaml 未找到 task-cloud-service，无法校验 installed-images 路由")


def validate_spa_catch_all_no_hosts(routes_doc: dict) -> None:
    """spa-catch-all 禁止绑定 hosts（OPT-20260809-006 静态门禁）。

    2026-08-09 修复回归（回归点 8f6194c 08-03）：spa-catch-all（priority 10,
    uri /*, GET/HEAD）绑定 hosts [daydaymoney.com, www.daydaymoney.com] 后，该域名下
    全部请求（含 /api/*）被 host 匹配分支优先命中并转发到 taskFE(4000) → 500/502。
    该回归静默存在 6 天（08-03→08-09），仅靠 routes.yaml 注释防护不可靠。

    本校验是生成链 SSOT 门禁：只要 spa-catch-all 声明 hosts 字段（含空列表）即
    硬失败，无论 --check/--lint/写出路径均无法绕过。
    """
    for rule in routes_doc.get("routes") or []:
        if rule.get("id") == "spa-catch-all" and "hosts" in rule:
            raise ValueError(
                "spa-catch-all 禁止绑定 hosts（OPT-20260809-006）：hosts 绑定会劫持该域名全部 "
                "GET/HEAD（含 /api/*），priority 10 在 host 匹配分支压过 priority 850 的业务路由 "
                "（如 taskauth-login）→ 被转发到 taskFE(4000) → 500/502。"
                "请移除 hosts 字段，由 priority 排序兜底：/api/* → api-orphaned/业务路由，非 API → SPA 壳。"
            )


def lint_routes_wildcard_coverage(routes_doc: dict, strict: bool = False) -> list[str]:
    """检查非通配路由是否遗漏 /* 子路径覆盖。

    规则：若路由有以 ``/`` 结尾的精确 URI（表明是集合资源端点），
    且无 ``*`` 通配符，则可能缺少对子路径（如 ``/resend/``、``/{id}/``）的覆盖。

    返回警告消息列表；strict=True 时将首个警告转为 ValueError。
    """
    warnings: list[str] = []
    catch_all_ids = {"django-default", "spa-catch-all"}

    for rule in routes_doc.get("routes") or []:
        rid = rule.get("id", "")
        if rid in catch_all_ids:
            continue

        uris: list[str] = rule.get("uris") or []
        if not uris:
            uri = rule.get("uri", "")
            if uri:
                uris = [uri]

        has_wildcard = any("*" in u for u in uris)
        if has_wildcard:
            continue

        # 检查是否有以 / 结尾的精确 URI（集合资源端点）
        collection_uris = [u for u in uris if u.endswith("/") and "*" not in u]
        if not collection_uris:
            continue

        upstream = rule.get("upstream", "?")
        priority = rule.get("priority", "?")
        msg = (
            f"[{rid}] upstream={upstream} priority={priority}: "
            f"缺少 /* 通配符 — 当前仅有精确匹配 {collection_uris}，"
            f"若上游存在子路径（如 /resend/、/{{id}}/）将回退到低优先级通配路由。"
            f"建议在 uris 中添加通配条目（如 {collection_uris[0]}*）。"
        )
        warnings.append(msg)

    if strict and warnings:
        raise ValueError(
            "路由通配符覆盖率门禁失败:\n" + "\n".join(f"  {w}" for w in warnings)
        )

    return warnings


def generate(check_only: bool = False, lint_strict: bool = False, allow_write: bool = True) -> str:
    routes_doc, conf = _load()
    validate_installed_images_route_to_task_cloud(routes_doc)
    # OPT-20260809-006: spa-catch-all hosts 绑定回归门禁 —— 在所有生成/校验路径最前
    # 执行，保证 --check/--lint/写出全部硬失败（防 2026-08-09 微信登录 502 回归）。
    validate_spa_catch_all_no_hosts(routes_doc)

    # 通配符覆盖率门禁（在代码生成前执行，发现问题时根据 strict 模式报错或警告）
    lint_warnings = lint_routes_wildcard_coverage(routes_doc, strict=lint_strict)
    if lint_warnings:
        for w in lint_warnings:
            print(f"  ⚠ {w}", file=sys.stderr)
        if not lint_strict:
            print(
                f"  ℹ 共 {len(lint_warnings)} 条通配符覆盖率建议（运行 --lint 查看详情）",
                file=sys.stderr,
            )
    upstream_defs = routes_doc.get("upstreams") or {}
    validate_upstream_docs_contract(upstream_defs)

    apisix_upstreams = []
    for name, upstream_def in upstream_defs.items():
        upstream_obj = {
            "id": f"up-{name}",
            "nodes": _upstream_nodes(upstream_defs, name, conf, upstream_def),
            "type": "roundrobin",
            "timeout": {"connect": 15, "send": 120, "read": 120},
        }
        upstream_obj.update(_upstream_retries(upstream_def))
        upstream_obj.update(_upstream_health_checks(upstream_def))
        apisix_upstreams.append(upstream_obj)
    apisix_upstreams.append(
        {
            "id": "up-deny",
            "nodes": {"127.0.0.1:1": 1},
            "type": "roundrobin",
        }
    )

    routes = [_build_route(r, upstream_defs, conf) for r in routes_doc.get("routes") or []]

    if _docs_enabled(conf):
        apisix_upstreams.append(
            {
                "id": "up-docsPortal",
                "nodes": _upstream_nodes(upstream_defs, "docsPortal", conf, {}),
                "type": "roundrobin",
                "timeout": {"connect": 5, "send": 30, "read": 30},
            }
        )
        docs_routes = _build_docs_routes(upstream_defs, conf)
        routes = docs_routes + routes
        if not check_only and allow_write:
            _write_portal_html(upstream_defs, conf)
            _write_ops_portal_html(upstream_defs, conf)

    global_plugins = {
        **_trace_plugin(),
        **_file_logger_plugin(),
        **_prometheus_plugin(),
        **_global_rate_limit(conf),
    }
    global_rules = [
        {
            "id": "global-observability",
            "plugins": global_plugins,
        }
    ]
    doc = {
        "upstreams": apisix_upstreams,
        "routes": routes,
        "global_rules": global_rules,
    }
    text = yaml.dump(doc, allow_unicode=True, sort_keys=False, default_flow_style=False)
    text += "#END\n"
    if check_only:
        existing = OUT.read_text(encoding="utf-8") if OUT.is_file() else ""
        if existing.strip() != text.strip():
            print("apisix.yaml is stale; run routes-apply", file=sys.stderr)
            sys.exit(1)
        expected_cors = _render_gatewaycors_go()
        existing_cors = GATEWAYCORS_OUT.read_text(encoding="utf-8") if GATEWAYCORS_OUT.is_file() else ""
        if existing_cors.strip() != expected_cors.strip():
            print(
                "shareLib/gatewaycors/allow_headers_gen.go is stale; run routes-to-apisix.py",
                file=sys.stderr,
            )
            sys.exit(1)
        return text
    if not allow_write:
        # OPT-20260809-006: --lint/--lint-strict 只校验不写文件 —— 此前 lint 会真正
        # 写出 apisix.yaml（注释称「不写文件」但代码未兑现），在无 docker 环境写出
        # loopback 上游节点 → bind-mount 热载 → 全站 502（与 OPT-20260806-026 同类）。
        return text
    OUT.parent.mkdir(parents=True, exist_ok=True)
    OUT.write_text(text, encoding="utf-8")
    _write_gatewaycors_go()
    return text


def main() -> None:
    parser = argparse.ArgumentParser()
    parser.add_argument("--check", action="store_true")
    parser.add_argument(
        "--lint",
        action="store_true",
        help="打印通配符覆盖率警告（非零退出码）",
    )
    parser.add_argument(
        "--lint-strict",
        action="store_true",
        help="通配符覆盖率门禁：缺通配符视为错误",
    )
    parser.add_argument(
        "--no-docker",
        action="store_true",
        help=(
            "显式声明 APISIX 运行在本机（非 Docker）：生成 127.0.0.1 上游节点。"
            "仅当你确知 standalone 监听文件运行在宿主机时使用；"
            "否则请通过 taskGateway/run.sh routes-apply 生成（其已导出 TASK_GATEWAY_APISIX_IN_DOCKER=1）。"
        ),
    )
    args = parser.parse_args()

    # OPT-20260806-026: 未显式声明运行模式时，默认写出 127.0.0.1 上游节点会在
    # APISIX standalone（Docker bind-mount 热载）下立即全站 502（2026-08-06 误触发过）。
    # 默认写入路径要求 TASK_GATEWAY_APISIX_IN_DOCKER=1（唯一入口 run.sh routes-apply）；
    # --check/--lint 不写文件，放行。
    writes_output = not (args.check or args.lint or args.lint_strict)
    if writes_output and not args.no_docker:
        in_docker = os.environ.get("TASK_GATEWAY_APISIX_IN_DOCKER", "").strip() in ("1", "true", "yes")
        if not in_docker:
            sys.exit(
                "ERROR: TASK_GATEWAY_APISIX_IN_DOCKER is not set — refusing to write "
                "apisix.yaml with loopback upstreams (APISIX standalone hot-reloads the "
                "bind-mounted file; wrong host => instant full-site 502).\n"
                "  Use: bash taskGateway/run.sh routes-apply   (exports the env var)\n"
                "  Or, if APISIX truly runs on the host, pass: --no-docker"
            )

    lint_strict = bool(args.lint_strict)
    # allow_write=writes_output：--check/--lint/--lint-strict 一律不写文件（lint 仅打印
    # 通配符覆盖率警告 + 运行 hosts 静态门禁），只有默认写出路径（run.sh routes-apply）
    # 才写 apisix.yaml 与 gatewaycors 代码生成物。
    generate(check_only=args.check, lint_strict=lint_strict, allow_write=writes_output)

    if args.lint or args.lint_strict:
        # lint 模式下不写输出文件
        return

    if not args.check:
        print(f"wrote {OUT}")


if __name__ == "__main__":
    main()
