"""Tests for BaseYAMLLoader domain service."""

import os
import tempfile

import pytest

from domain.services.base_yaml_loader import BaseYAMLLoader
from domain.value_objects.deploy_mode import DeployMode


class TestBaseYAMLLoader:
    @pytest.fixture
    def loader(self):
        return BaseYAMLLoader()

    @pytest.fixture
    def base_yaml_domain(self):
        return """\
scheme: https
baseDomain: daydaymoney.com
subdomains:
  api: api.${baseDomain}
  auth: auth.api.${baseDomain}
  gitoauth: gitoauth.api.${baseDomain}
  provider: provider.${baseDomain}
  gitlab: gitlab.${baseDomain}
  www: www.${baseDomain}
  base: ${baseDomain}
  aiendpoint: aiendpoint.${baseDomain}
  credential: credential.api.${baseDomain}
  agentsupport: agentsupport.api.${baseDomain}
  cloud: cloud.api.${baseDomain}
  relaytotrae: relay.${baseDomain}
  mockruncontainer: mockruncontainer.${baseDomain}
  businessapi: businessapi.${baseDomain}
  kafka: kafka.${baseDomain}
  gateway: ${scheme}://api.${baseDomain}
"""

    def _write_temp(self, content: str) -> str:
        tmp = tempfile.NamedTemporaryFile(mode="w", suffix=".yaml", delete=False)
        tmp.write(content)
        tmp.close()
        return tmp.name

    def test_overlays_conf_local_base_yaml(self, loader, base_yaml_domain, tmp_path):
        conf = tmp_path / "conf"
        conf.mkdir()
        (conf / "base.yaml").write_text(base_yaml_domain)
        local = tmp_path / "conf-local"
        local.mkdir()
        (local / "base.yaml").write_text("baseDomain: from-conf-local.test\n")
        scheme = loader.load(conf / "base.yaml")
        assert scheme.base_domain == "from-conf-local.test"
        assert scheme.resolve("api") == "api.from-conf-local.test"
        assert scheme.resolve("gateway") == "https://api.from-conf-local.test"

    def test_load_domain(self, loader, base_yaml_domain):
        path = self._write_temp(base_yaml_domain)
        try:
            scheme = loader.load(path)
            assert scheme.mode == DeployMode("domain")
            assert scheme.scheme == "https"
            assert scheme.resolve("api") == "api.daydaymoney.com"
            assert scheme.resolve("gateway") == "https://api.daydaymoney.com"
            assert scheme.base_domain == "daydaymoney.com"
            assert scheme.resolve_template("${scheme}://${subdomains.www}") == "https://www.daydaymoney.com"
        finally:
            os.unlink(path)

    def test_load_ignores_legacy_mode_local(self, loader):
        """Legacy mode/local sections are ignored; domain section wins."""
        content = """\
mode: local
scheme: https
baseDomain: example.com
subdomains:
  api: api.${baseDomain}
  auth: auth.api.${baseDomain}
  gitoauth: gitoauth.api.${baseDomain}
  provider: provider.${baseDomain}
  gitlab: gitlab.${baseDomain}
  www: www.${baseDomain}
  base: ${baseDomain}
  aiendpoint: aiendpoint.${baseDomain}
  credential: credential.api.${baseDomain}
  agentsupport: agentsupport.api.${baseDomain}
  cloud: cloud.api.${baseDomain}
  relaytotrae: relay.${baseDomain}
  mockruncontainer: mockruncontainer.${baseDomain}
  businessapi: businessapi.${baseDomain}
  kafka: kafka.${baseDomain}
  gateway: ${scheme}://api.${baseDomain}
local:
  api: 10.0.0.1:8001
  gateway: http://10.0.0.1:18081
"""
        path = self._write_temp(content)
        try:
            scheme = loader.load(path)
            assert scheme.mode == DeployMode("domain")
            assert scheme.resolve("api") == "api.daydaymoney.com"
            assert scheme.resolve("gateway") == "https://api.daydaymoney.com"
        finally:
            os.unlink(path)

    def test_base_domain_env_override(self, loader, base_yaml_domain):
        path = self._write_temp(base_yaml_domain)
        try:
            os.environ["BASE_DOMAIN"] = "override.io"
            scheme = loader.load(path)
            assert scheme.base_domain == "override.io"
            assert scheme.resolve("api") == "api.override.io"
            assert scheme.resolve("gateway") == "https://api.override.io"
        finally:
            os.environ.pop("BASE_DOMAIN", None)
            os.unlink(path)

    def test_scheme_env_override(self, loader, base_yaml_domain):
        path = self._write_temp(base_yaml_domain)
        try:
            os.environ["PUBLIC_SCHEME"] = "http"
            scheme = loader.load(path)
            assert scheme.scheme == "http"
            assert scheme.resolve("gateway") == "http://api.daydaymoney.com"
            assert scheme.resolve_template("${scheme}://${baseDomain}") == "http://daydaymoney.com"
        finally:
            os.environ.pop("PUBLIC_SCHEME", None)
            os.unlink(path)

    def test_base_domain_template_default(self, loader):
        content = """\
scheme: ${PUBLIC_SCHEME:-https}
baseDomain: ${BASE_DOMAIN:-daydaymoney.com}
subdomains:
  api: api.${baseDomain}
  auth: auth.api.${baseDomain}
  gitoauth: gitoauth.api.${baseDomain}
  provider: provider.${baseDomain}
  gitlab: gitlab.${baseDomain}
  www: www.${baseDomain}
  base: ${baseDomain}
  aiendpoint: aiendpoint.${baseDomain}
  credential: credential.api.${baseDomain}
  agentsupport: agentsupport.api.${baseDomain}
  cloud: cloud.api.${baseDomain}
  relaytotrae: relay.${baseDomain}
  mockruncontainer: mockruncontainer.${baseDomain}
  businessapi: businessapi.${baseDomain}
  kafka: kafka.${baseDomain}
  gateway: ${scheme}://api.${baseDomain}
"""
        path = self._write_temp(content)
        try:
            os.environ.pop("BASE_DOMAIN", None)
            os.environ.pop("PUBLIC_SCHEME", None)
            scheme = loader.load(path)
            assert scheme.base_domain == "example.com"
            assert scheme.scheme == "https"
            assert scheme.resolve("gateway") == "https://api.daydaymoney.com"
        finally:
            os.unlink(path)

    def test_missing_file_raises(self, loader):
        with pytest.raises(FileNotFoundError, match="conf/base.yaml not found"):
            loader.load("/nonexistent/path/base.yaml")

    def test_missing_subdomains_raises(self, loader):
        content = "baseDomain: test.com\n"
        path = self._write_temp(content)
        try:
            with pytest.raises(ValueError, match="subdomains must be a mapping"):
                loader.load(path)
        finally:
            os.unlink(path)

    def test_missing_required_subdomain_key_raises(self, loader):
        content = """\
baseDomain: test.com
subdomains:
  api: api.${baseDomain}
"""
        path = self._write_temp(content)
        try:
            with pytest.raises(ValueError, match="subdomains.auth is required"):
                loader.load(path)
        finally:
            os.unlink(path)

    def test_invalid_scheme_raises(self, loader):
        content = """\
scheme: ftp
baseDomain: test.com
subdomains:
  api: api.${baseDomain}
  auth: auth.api.${baseDomain}
  gitoauth: gitoauth.api.${baseDomain}
  provider: provider.${baseDomain}
  gitlab: gitlab.${baseDomain}
  www: www.${baseDomain}
  base: ${baseDomain}
  aiendpoint: aiendpoint.${baseDomain}
  credential: credential.api.${baseDomain}
  agentsupport: agentsupport.api.${baseDomain}
  cloud: cloud.api.${baseDomain}
  relaytotrae: relay.${baseDomain}
  mockruncontainer: mockruncontainer.${baseDomain}
  businessapi: businessapi.${baseDomain}
  kafka: kafka.${baseDomain}
  gateway: ${scheme}://api.${baseDomain}
"""
        path = self._write_temp(content)
        try:
            with pytest.raises(ValueError, match="scheme must be http or https"):
                loader.load(path)
        finally:
            os.unlink(path)
