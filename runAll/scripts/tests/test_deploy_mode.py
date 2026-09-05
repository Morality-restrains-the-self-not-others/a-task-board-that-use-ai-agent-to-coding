"""Tests for DeployMode value object."""

import pytest

from domain.value_objects.deploy_mode import DeployMode


class TestDeployMode:
    def test_domain_mode(self):
        mode = DeployMode("domain")
        assert mode.value == "domain"
        assert mode.is_domain is True
        assert mode.is_local is False

    def test_local_mode(self):
        mode = DeployMode("local")
        assert mode.value == "local"
        assert mode.is_local is True
        assert mode.is_domain is False

    def test_case_insensitive(self):
        assert DeployMode("DOMAIN").value == "domain"
        assert DeployMode("Local").value == "local"

    def test_strips_whitespace(self):
        assert DeployMode("  domain  ").value == "domain"

    def test_invalid_value_raises(self):
        with pytest.raises(ValueError, match="DEPLOY_MODE must be one of"):
            DeployMode("invalid")
        with pytest.raises(ValueError, match="DEPLOY_MODE must be one of"):
            DeployMode("")
        with pytest.raises(ValueError, match="DEPLOY_MODE must be one of"):
            DeployMode("production")

    def test_equality(self):
        assert DeployMode("domain") == DeployMode("domain")
        assert DeployMode("local") == DeployMode("local")
        assert DeployMode("domain") != DeployMode("local")

    def test_hash(self):
        assert hash(DeployMode("domain")) == hash(DeployMode("domain"))
        assert len({DeployMode("domain"), DeployMode("domain"), DeployMode("local")}) == 2

    def test_repr(self):
        assert repr(DeployMode("domain")) == "DeployMode('domain')"
        assert str(DeployMode("local")) == "local"
