"""Tests for AddressingScheme value object."""

import pytest

from domain.value_objects.addressing_scheme import AddressingScheme
from domain.value_objects.deploy_mode import DeployMode


class TestAddressingScheme:
    @pytest.fixture
    def domain_scheme(self):
        return AddressingScheme(
            mode=DeployMode("domain"),
            addresses={
                "api": "api.daydaymoney.com",
                "auth": "auth.api.daydaymoney.com",
                "gateway": "daydaymoney.com",
                "www": "www.daydaymoney.com",
            },
            base_domain="daydaymoney.com",
        )

    @pytest.fixture
    def local_scheme(self):
        return AddressingScheme(
            mode=DeployMode("local"),
            addresses={
                "api": "183.250.1.132:8001",
                "auth": "183.250.1.132:8003",
                "gateway": "183.250.1.132:18081",
                "www": "183.250.1.132:4000",
            },
            base_domain="183.250.1.132",
        )

    def test_requires_non_empty_addresses(self):
        with pytest.raises(ValueError, match="addresses must not be empty"):
            AddressingScheme(DeployMode("local"), {}, "127.0.0.1")

    def test_requires_deploy_mode_type(self):
        with pytest.raises(TypeError):
            AddressingScheme("domain", {"api": "x"}, "x")  # type: ignore

    def test_resolve_existing_key(self, domain_scheme):
        assert domain_scheme.resolve("api") == "api.daydaymoney.com"

    def test_resolve_missing_key_raises(self, domain_scheme):
        with pytest.raises(KeyError, match="addressing key 'unknown'"):
            domain_scheme.resolve("unknown")

    def test_resolve_template_subdomains(self, domain_scheme):
        result = domain_scheme.resolve_template("http://${subdomains.api}/health")
        assert result == "http://api.daydaymoney.com/health"

    def test_resolve_template_scheme(self, domain_scheme):
        result = domain_scheme.resolve_template("${scheme}://${subdomains.www}")
        assert result == "https://www.daydaymoney.com"

    def test_resolve_template_base_domain(self, domain_scheme):
        result = domain_scheme.resolve_template("${scheme}://${baseDomain}")
        assert result == "https://daydaymoney.com"

    def test_resolve_template_multiple(self, domain_scheme):
        result = domain_scheme.resolve_template(
            "${subdomains.api} -> ${subdomains.auth} -> ${subdomains.gateway}"
        )
        assert result == "api.daydaymoney.com -> auth.api.daydaymoney.com -> daydaymoney.com"

    def test_resolve_template_no_match_passthrough(self, domain_scheme):
        result = domain_scheme.resolve_template("smtp.qq.com:465")
        assert result == "smtp.qq.com:465"

    def test_resolve_template_local_mode(self, local_scheme):
        result = local_scheme.resolve_template("http://${subdomains.api}/health")
        assert result == "http://183.250.1.132:8001/health"

    def test_equality(self, domain_scheme):
        same = AddressingScheme(
            mode=DeployMode("domain"),
            addresses={"api": "api.daydaymoney.com", "auth": "auth.api.daydaymoney.com",
                       "gateway": "daydaymoney.com", "www": "www.daydaymoney.com"},
            base_domain="daydaymoney.com",
        )
        assert domain_scheme == same

    def test_immutability_addresses(self, domain_scheme):
        addrs = domain_scheme.addresses
        addrs["new"] = "test"
        assert "new" not in domain_scheme.addresses

    def test_mode_property(self, domain_scheme, local_scheme):
        assert domain_scheme.mode.is_domain
        assert local_scheme.mode.is_local

    def test_repr(self, domain_scheme):
        r = repr(domain_scheme)
        assert "AddressingScheme" in r
        assert "api.daydaymoney.com" in r
