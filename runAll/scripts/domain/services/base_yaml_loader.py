"""BaseYAMLLoader domain service — reads conf/base.yaml into AddressingScheme."""

from __future__ import annotations

import os
import re
from pathlib import Path
from typing import Dict

from ..value_objects.addressing_scheme import AddressingScheme
from ..value_objects.deploy_mode import DeployMode

# Pattern: ${VAR:-default} only (bare ${VAR} left for dedicated resolvers).
_ENV_DEFAULT_RE = re.compile(r"\$\{(\w+):-([^}]*)\}")


def _expand_env_default(value: str) -> str:
    """Expand ${VAR:-default} patterns in a string value.

    Only resolves WITH a :-default clause. Bare ${VAR} is left unchanged.
    Returns the env var value if set, otherwise the default part.
    """
    def _replace(match):
        var_name = match.group(1)
        default = match.group(2)
        env_val = os.environ.get(var_name)
        if env_val is not None:
            return env_val
        return default
    return _ENV_DEFAULT_RE.sub(_replace, value)


def _normalize_scheme(raw: str) -> str:
    scheme = str(raw).strip().lower().rstrip(":/")
    if scheme not in ("http", "https"):
        raise ValueError(f"base.yaml: scheme must be http or https, got {raw!r}")
    return scheme


class BaseYAMLLoader:
    """Domain service: load conf/base.yaml and construct AddressingScheme.

    Rules:
    - Domain addressing only: expands subdomains.* with scheme/baseDomain substitution
    - Missing base.yaml → FileNotFoundError (no graceful degradation)
    - Legacy mode/local keys in base.yaml are ignored
    """

    # Service keys expected in subdomains
    _SERVICE_KEYS = (
        "api",
        "auth",
        "gitoauth",
        "provider",
        "gitlab",
        "www",
        "base",
        "aiendpoint",
        "credential",
        "agentsupport",
        "cloud",
        "relaytotrae",
        "mockruncontainer",
        "businessapi",
        "kafka",
        "gateway",
    )

    def __init__(self, yaml_loader=None) -> None:
        """Inject yaml_loader for testability. Defaults to conf_lib.load_yaml."""
        self._load_yaml = yaml_loader or self._default_yaml_loader

    @staticmethod
    def _default_yaml_loader(path: Path) -> dict:
        from conf_lib import load_yaml
        return load_yaml(path)

    def load(self, base_yaml_path: str | Path) -> AddressingScheme:
        """Load base.yaml and return AddressingScheme.

        Args:
            base_yaml_path: Path to conf/base.yaml

        Returns:
            AddressingScheme with domain mode and resolved addresses

        Raises:
            FileNotFoundError: base.yaml not found
            ValueError: missing required fields
        """
        path = Path(base_yaml_path)
        if not path.is_file():
            raise FileNotFoundError(
                f"conf/base.yaml not found at {path} — "
                f"this file is required for service addressing"
            )

        base = self._load_yaml(path)
        if not isinstance(base, dict):
            raise ValueError(f"base.yaml must contain a mapping, got {type(base)}")
        # ADR-0054: overlay conf-local/base.yaml without replacing injected yaml_loader.
        resolved = Path(path).resolve()
        if resolved.name == "base.yaml" and resolved.parent.name == "conf":
            from conf_local import merge_conf_local  # noqa: PLC0415

            base = merge_conf_local(resolved.parent.parent, "base.yaml", base)

        return self._resolve_domain(base)

    def _resolve_domain(self, base: dict) -> AddressingScheme:
        """Expand subdomains section for domain addressing."""
        scheme_raw = base.get("scheme", "https")
        scheme = os.environ.get("PUBLIC_SCHEME") or _expand_env_default(str(scheme_raw))
        if not scheme:
            scheme = "https"
        scheme = _normalize_scheme(scheme)

        base_domain_raw = base.get("baseDomain", "")
        base_domain = os.environ.get("BASE_DOMAIN") or _expand_env_default(str(base_domain_raw))
        if not base_domain:
            raise ValueError("base.yaml: baseDomain is required")

        subs = base.get("subdomains")
        if not isinstance(subs, dict):
            raise ValueError("base.yaml: subdomains must be a mapping")

        # Resolve infraHost (may reference ${INFRA_HOST:-default}).
        infra_host_raw = base.get("infraHost", "")
        infra_host = _expand_env_default(str(infra_host_raw)) if infra_host_raw else ""

        addresses: Dict[str, str] = {}
        for key in self._SERVICE_KEYS:
            template = subs.get(key)
            if not template:
                raise ValueError(
                    f"base.yaml: subdomains.{key} is required"
                )
            addr = (
                str(template)
                .replace("${scheme}", scheme)
                .replace("${baseDomain}", str(base_domain))
            )
            if infra_host:
                addr = addr.replace("${infraHost}", infra_host)
            addresses[key] = addr

        return AddressingScheme(
            mode=DeployMode("domain"),
            addresses=addresses,
            base_domain=str(base_domain),
            scheme=scheme,
        )
