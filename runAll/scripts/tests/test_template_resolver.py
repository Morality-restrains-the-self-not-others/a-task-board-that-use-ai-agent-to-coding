"""Tests for TemplateResolver domain service."""

import pytest

from domain.services.template_resolver import TemplateResolver
from domain.value_objects.addressing_scheme import AddressingScheme
from domain.value_objects.deploy_mode import DeployMode


class TestTemplateResolver:
    @pytest.fixture
    def scheme(self):
        return AddressingScheme(
            mode=DeployMode("domain"),
            addresses={
                "api": "api.daydaymoney.com",
                "auth": "auth.api.daydaymoney.com",
                "gateway": "daydaymoney.com",
                "www": "www.daydaymoney.com",
                "gitoauth": "gitoauth.api.daydaymoney.com",
            },
            base_domain="daydaymoney.com",
        )

    def test_resolve_flat_dict(self, scheme):
        config = {
            "host": "127.0.0.1",
            "publicBase": "http://${subdomains.gateway}",
            "apiBaseUrl": "https://${subdomains.api}",
        }
        result = TemplateResolver.resolve(config, scheme)
        assert result["host"] == "127.0.0.1"
        assert result["publicBase"] == "http://daydaymoney.com"
        assert result["apiBaseUrl"] == "https://api.daydaymoney.com"

    def test_resolve_nested_dict(self, scheme):
        config = {
            "oidc": {
                "issuer": "http://${subdomains.gateway}",
                "redirectUri": "https://${subdomains.auth}/callback",
            }
        }
        result = TemplateResolver.resolve(config, scheme)
        assert result["oidc"]["issuer"] == "http://daydaymoney.com"
        assert result["oidc"]["redirectUri"] == "https://auth.api.daydaymoney.com/callback"

    def test_resolve_list(self, scheme):
        config = {
            "allowedOrigins": [
                "http://${subdomains.www}",
                "https://${subdomains.gateway}",
            ]
        }
        result = TemplateResolver.resolve(config, scheme)
        assert result["allowedOrigins"] == [
            "http://www.daydaymoney.com",
            "https://daydaymoney.com",
        ]

    def test_resolve_base_domain(self, scheme):
        config = {"allowedHost": "http://${baseDomain}"}
        result = TemplateResolver.resolve(config, scheme)
        assert result["allowedHost"] == "http://daydaymoney.com"

    def test_non_string_values_preserved(self, scheme):
        config = {"port": 8001, "enabled": True, "timeout": None, "rate": 1.5}
        result = TemplateResolver.resolve(config, scheme)
        assert result == config  # unchanged

    def test_does_not_mutate_original(self, scheme):
        config = {"url": "http://${subdomains.api}"}
        _ = TemplateResolver.resolve(config, scheme)
        assert config["url"] == "http://${subdomains.api}"  # unchanged

    def test_no_template_vars_returns_equal(self, scheme):
        config = {"host": "127.0.0.1", "port": 8001}
        result = TemplateResolver.resolve(config, scheme)
        assert result == {"host": "127.0.0.1", "port": 8001}

    def test_unknown_subdomain_passthrough(self, scheme):
        config = {"url": "${subdomains.unknown}"}
        result = TemplateResolver.resolve(config, scheme)
        assert result["url"] == "${subdomains.unknown}"
