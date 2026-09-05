"""DeployMode value object — domain | local."""

from __future__ import annotations


class DeployMode:
    """DeployMode value object.

    Immutable. Two valid values: "domain" or "local".
    Constructed from base.yaml mode field or DEPLOY_MODE env var.
    No default constructor — must be built from explicit value.
    """

    DOMAIN = "domain"
    LOCAL = "local"
    _VALID = frozenset({DOMAIN, LOCAL})

    def __init__(self, value: str) -> None:
        v = str(value).strip().lower()
        if v not in self._VALID:
            raise ValueError(
                f"DEPLOY_MODE must be one of {sorted(self._VALID)}, got {value!r}"
            )
        self._value: str = v

    @property
    def value(self) -> str:
        return self._value

    @property
    def is_domain(self) -> bool:
        return self._value == self.DOMAIN

    @property
    def is_local(self) -> bool:
        return self._value == self.LOCAL

    def __eq__(self, other: object) -> bool:
        if not isinstance(other, DeployMode):
            return False
        return self._value == other._value

    def __hash__(self) -> int:
        return hash(self._value)

    def __repr__(self) -> str:
        return f"DeployMode({self._value!r})"

    def __str__(self) -> str:
        return self._value
