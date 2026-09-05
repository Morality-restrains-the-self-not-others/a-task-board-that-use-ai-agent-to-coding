"""AddressingScheme value object — resolved service address map."""

from __future__ import annotations

import os
import re
from typing import Dict

from .deploy_mode import DeployMode

# Pattern: ${VAR:-default} only (bare ${VAR} without default is left as-is
# for dedicated resolvers like routes-to-apisix _expand_infra_host_template).
_ENV_DEFAULT_RE = re.compile(r"\$\{(\w+):-([^}]*)\}")


def _expand_env_default(value: str) -> str:
    """Expand ${VAR:-default} patterns in a string value.

    Only resolves patterns WITH a :-default clause. Bare ${VAR} is left
    unchanged — callers with dedicated VAR resolvers must handle those.
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


class AddressingScheme:
    """AddressingScheme value object.

    Immutable map of service keys → resolved addresses.
    Constructed by BaseYAMLLoader from conf/base.yaml.
    No default constructor — base.yaml is the single source of truth.
    """

    def __init__(
        self,
        mode: DeployMode,
        addresses: Dict[str, str],
        base_domain: str,
        scheme: str = "https",
    ) -> None:
        if not isinstance(mode, DeployMode):
            raise TypeError(f"mode must be DeployMode, got {type(mode)}")
        if not isinstance(addresses, dict):
            raise TypeError(f"addresses must be dict, got {type(addresses)}")
        if not addresses:
            raise ValueError("addresses must not be empty")
        scheme_norm = str(scheme).strip().lower().rstrip(":/")
        if scheme_norm not in ("http", "https"):
            raise ValueError(f"scheme must be http or https, got {scheme!r}")
        self._mode = mode
        self._addresses: Dict[str, str] = dict(addresses)
        self._base_domain: str = str(base_domain)
        self._scheme: str = scheme_norm

    @property
    def mode(self) -> DeployMode:
        return self._mode

    @property
    def base_domain(self) -> str:
        return self._base_domain

    @property
    def scheme(self) -> str:
        return self._scheme

    def resolve(self, key: str) -> str:
        """Return the resolved address for a service key.

        Raises KeyError if the key is not in the addressing map.
        """
        if key not in self._addresses:
            raise KeyError(f"addressing key {key!r} not found; available: {sorted(self._addresses)}")
        return self._addresses[key]

    def resolve_template(self, value: str) -> str:
        """Replace ${scheme}, ${subdomains.xxx}, ${baseDomain}, ${infraHost},
        and ${VAR:-default} env-var patterns in a string value."""
        result = value
        for key, addr in self._addresses.items():
            result = result.replace(f"${{subdomains.{key}}}", addr)
        result = result.replace("${baseDomain}", self._base_domain)
        result = result.replace("${scheme}", self._scheme)
        # Fallback: resolve remaining ${VAR:-default} env-var patterns
        # (e.g. ${INFRA_HOST:-10.2.150.68} if not already resolved via subdomains).
        result = _expand_env_default(result)
        return result

    @property
    def addresses(self) -> Dict[str, str]:
        return dict(self._addresses)

    def __eq__(self, other: object) -> bool:
        if not isinstance(other, AddressingScheme):
            return False
        return (
            self._mode == other._mode
            and self._addresses == other._addresses
            and self._base_domain == other._base_domain
            and self._scheme == other._scheme
        )

    def __hash__(self) -> int:
        return hash(
            (
                self._mode,
                tuple(sorted(self._addresses.items())),
                self._base_domain,
                self._scheme,
            )
        )

    def __repr__(self) -> str:
        return (
            f"AddressingScheme(mode={self._mode}, addresses={self._addresses}, "
            f"base_domain={self._base_domain!r}, scheme={self._scheme!r})"
        )
