#!/usr/bin/env python3
"""GITLAB_OMNIBUS_CONFIG must use nginx on/off, not Ruby true (GitLab 19.2)."""
from __future__ import annotations

from pathlib import Path

COMPOSE = Path(__file__).resolve().parent.parent / "docker-compose.yml"


def test_real_ip_recursive_is_on_not_true():
    text = COMPOSE.read_text(encoding="utf-8")
    assert "nginx['real_ip_recursive'] = 'on'" in text, text
    assert "nginx['real_ip_recursive'] = true" not in text


def test_trusted_proxy_lists_omit_unquoted_ipv6_loopback():
    """Omnibus dumps ::1 unquoted; Psych loads it as Symbol :1 and Puma dies."""
    text = COMPOSE.read_text(encoding="utf-8")
    assert "nginx['real_ip_trusted_addresses'] = ['127.0.0.0/8', '172.16.0.0/12']" in text
    assert "gitlab_workhorse['trusted_cidrs'] = ['127.0.0.0/8', '172.16.0.0/12']" in text
    assert "'::1'" not in text


if __name__ == "__main__":
    test_real_ip_recursive_is_on_not_true()
    test_trusted_proxy_lists_omit_unquoted_ipv6_loopback()
    print("ok test_real_ip_recursive_is_on_not_true")
    print("ok test_trusted_proxy_lists_omit_unquoted_ipv6_loopback")
