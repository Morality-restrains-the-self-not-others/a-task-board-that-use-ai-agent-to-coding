"""TemplateResolver domain service — resolve ${subdomains.xxx} in config values."""

from __future__ import annotations

from typing import Any

from ..value_objects.addressing_scheme import AddressingScheme


class TemplateResolver:
    """Domain service: recursively resolve template variables in a config dict.

    Pure function — no side effects, no global state.
    Returns a NEW dict; does not mutate the input.

    Replaces:
      - ${subdomains.xxx} → scheme.resolve("xxx")
      - ${baseDomain}    → scheme.base_domain
    """

    @staticmethod
    def resolve(config: dict[str, Any], scheme: AddressingScheme) -> dict[str, Any]:
        """Recursively resolve template variables.

        Args:
            config: Configuration dict with possible ${subdomains.xxx} placeholders
            scheme: Resolved AddressingScheme

        Returns:
            New dict with all template variables replaced
        """
        return TemplateResolver._walk(config, scheme)

    @staticmethod
    def _walk(obj: Any, scheme: AddressingScheme) -> Any:
        if isinstance(obj, dict):
            return {k: TemplateResolver._walk(v, scheme) for k, v in obj.items()}
        if isinstance(obj, list):
            return [TemplateResolver._walk(v, scheme) for v in obj]
        if isinstance(obj, str):
            return scheme.resolve_template(obj)
        return obj
