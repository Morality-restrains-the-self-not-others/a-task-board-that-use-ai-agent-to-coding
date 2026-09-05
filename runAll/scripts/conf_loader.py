"""Load monorepo conf/<app>/ YAML — self-contained config loader for runAll.

Replaces the previous dependency on task2app/Saas_project/config/conf_loader.py.
Uses conf_lib shared helpers for YAML loading, deep merge, and repo root detection.
"""
from __future__ import annotations

import sys
from functools import lru_cache
from pathlib import Path
from typing import Any

# Import shared helpers from the same directory
_here = Path(__file__).resolve().parent
if str(_here) not in sys.path:
    sys.path.insert(0, str(_here))

from conf_lib import deep_merge, load_yaml, repo_root as _repo_root  # noqa: E402
from conf_local import merge_conf_local  # noqa: E402
from domain.services.base_yaml_loader import BaseYAMLLoader
from domain.services.template_resolver import TemplateResolver

JSON_KEY_TO_APP_DIR = {
    # OPT-20260806-057: django 目录退役，django 键移除，新增语义化目录映射
    "sms": "core/sms",
    "email": "core/email",
    "sso": "core/sso",
    "vue": "frontend/vue",
    "aiProvider": "ai/ai-provider",
    "gitService": "infra/git-service",
    "dockerInfra": "infra/docker-infra",
    "mock_trae_worker": "mock/mock-trae-worker",
    "taskSSE": "gateway/task-sse",
    "task-sse": "gateway/task-sse",
    "taskAgentSupport": "ai/task-agent-support",
    "taskAIEndPoint": "ai/task-ai-endpoint",
    "taskContainerGateway": "gateway/task-container-gateway",
    "taskGateway": "gateway/task-gateway",
    "relayToTrae": "infra/relay-to-trae",
    "taskAuth": "auth/task-auth",
    "taskBill": "billing/task-bill",
    "stripe": "billing/stripe",
    "domainEvents": "events/domain-events",
}


def monorepo_root() -> Path:
    """Delegates to conf_lib.repo_root for consistent marker-based detection."""
    return _repo_root()


@lru_cache(maxsize=1)
def _load_addressing_scheme():
    """Load and cache AddressingScheme from conf/base.yaml."""
    root = monorepo_root()
    base_path = root / "conf" / "base.yaml"
    loader = BaseYAMLLoader()
    return loader.load(str(base_path))


def resolve_app_config_dir(app_dir: str) -> str:
    """Map legacy JSON keys (``sms``, ``vue``, …) to ``conf/<path>`` directories.
    OPT-20260806-057: django 键退役，语义化目录（sms/email/sso）映射。"""
    key = (app_dir or "").strip().replace("\\", "/").strip("/")
    return JSON_KEY_TO_APP_DIR.get(key, key)


def _merge_fragment(root: Path, app_dir: str, name: str, cfg: dict[str, Any]) -> dict[str, Any]:
    """Merge tracked conf/<app>/<name> then conf-local/<app>/<name> (ADR-0054)."""
    rel = f"{app_dir}/{name}"
    frag_path = root / "conf" / app_dir / name
    frag: dict[str, Any] = {}
    if frag_path.is_file():
        loaded = load_yaml(frag_path)
        if isinstance(loaded, dict):
            frag = loaded
    frag = merge_conf_local(root, rel, frag)
    if frag:
        return deep_merge(cfg, frag)
    return cfg


def load_app_config(app_dir: str, *, include_test: bool = False) -> dict[str, Any]:
    app_dir = resolve_app_config_dir(app_dir)
    root = monorepo_root()
    app_path = root / "conf" / app_dir
    cfg_path = app_path / "config.yaml"
    if not cfg_path.is_file():
        return {}
    cfg = load_yaml(cfg_path)
    cfg = merge_conf_local(root, f"{app_dir}/config.yaml", cfg)
    cfg = _merge_fragment(root, app_dir, "docker-infra.yaml", cfg)
    cfg = _merge_fragment(root, app_dir, "git-service.yaml", cfg)
    if include_test:
        test = app_path / "config.test.yaml"
        if test.is_file():
            cfg = deep_merge(cfg, load_yaml(test))
    # Resolve ${subdomains.xxx} / ${baseDomain} template variables
    scheme = _load_addressing_scheme()
    cfg = TemplateResolver.resolve(cfg, scheme)
    return cfg


def load_git_oauth_catalog() -> dict[str, Any]:
    root = monorepo_root()
    prov_dir = root / "conf" / "auth" / "git-oauth" / "providers"
    catalog: dict[str, Any] = {}
    if not prov_dir.is_dir():
        return catalog
    scheme = _load_addressing_scheme()
    for path in sorted(prov_dir.glob("*.yaml")):
        block = load_yaml(path)
        rel = path.relative_to(root / "conf").as_posix()
        block = merge_conf_local(root, rel, block)
        block = TemplateResolver.resolve(block, scheme)
        # Use provider:service_provider as unique key to avoid collisions
        # when different providers resolve to the same website URL.
        provider = str(block.get("provider") or "").strip().lower()
        service_provider = str(block.get("service_provider") or "").strip().lower()
        if provider and service_provider:
            key = f"{provider}:{service_provider}"
        elif provider:
            key = provider
        else:
            target = block.get("target") if isinstance(block.get("target"), dict) else {}
            website = target.get("website")
            key = str(website) if website else path.stem
        catalog[key] = block
    return catalog


def load_domain_events_config() -> dict[str, Any]:
    root = monorepo_root()
    de_dir = root / "conf" / "events" / "domain-events"
    base = load_yaml(de_dir / "config.yaml")
    base = merge_conf_local(root, "events/domain-events/config.yaml", base)
    base = _merge_fragment(root, "events/domain-events", "docker-infra.yaml", base)
    consumers: dict[str, Any] = {}
    for child in sorted(de_dir.iterdir()):
        if not child.is_dir():
            continue
        cfg_path = child / "config.yaml"
        if cfg_path.is_file():
            consumers[child.name] = merge_conf_local(
                root,
                f"events/domain-events/{child.name}/config.yaml",
                load_yaml(cfg_path),
            )
    if consumers:
        base["consumers"] = consumers
    return base


@lru_cache(maxsize=1)
def build_runtime_snapshot() -> dict[str, Any]:
    """Aggregate conf/<app>/ into former port_config.json top-level shape."""
    out: dict[str, Any] = {}
    # OPT-20260806-057: django 目录退役，不再产出 django 键
    vue = load_app_config("frontend/vue")
    out["vue"] = vue
    out["gitOauth"] = load_git_oauth_catalog()
    out["domainEvents"] = load_domain_events_config()
    for json_key, app_dir in JSON_KEY_TO_APP_DIR.items():
        if json_key in ("domainEvents", "gitOauth", "vue"):
            continue
        block = load_app_config(app_dir)
        if block:
            out[json_key] = block
    # OPT-20260806-057: django 目录退役，sso 配置从 conf/core/sso.yaml 读取
    sso = load_app_config("core/sso")
    if sso.get("ssoJwtSecret"):
        out["task2appSsoJwtSecret"] = sso["ssoJwtSecret"]
    if sso.get("ssh_login_allowed_addresses") is not None:
        out["ssh_login_allowed_addresses"] = sso["ssh_login_allowed_addresses"]
    # Inject resolved addressing scheme for consumers
    scheme = _load_addressing_scheme()
    out["_addressing"] = {
        "mode": scheme.mode.value,
        "base_domain": scheme.base_domain,
        "scheme": scheme.scheme,
        "addresses": scheme.addresses,
    }
    return out
