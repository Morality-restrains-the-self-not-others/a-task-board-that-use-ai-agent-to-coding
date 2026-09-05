#!/usr/bin/env python3
"""ensure-edge-tunnels health predicates (no SSH)."""

from __future__ import annotations

import importlib.util
import unittest
from pathlib import Path


def load_mod():
    path = Path(__file__).resolve().parents[1] / "edge_tunnel_health.py"
    spec = importlib.util.spec_from_file_location("edge_tunnel_health", path)
    assert spec and spec.loader
    mod = importlib.util.module_from_spec(spec)
    spec.loader.exec_module(mod)
    return mod


class EdgeTunnelHealthTest(unittest.TestCase):
    def setUp(self):
        self.mod = load_mod()

    def test_real_autossh_cmdline_matches(self):
        cmd = (
            "/usr/lib/autossh/autossh -M 0 -N -o ServerAliveInterval 30 "
            "-R 8010:127.0.0.1:8010 sh"
        )
        self.assertTrue(self.mod.cmdline_is_real_autossh(cmd, "", "8010"))

    def test_agent_bash_wrapper_does_not_match(self):
        # 2026-09-01: pgrep -f 'autossh.*-R 8010:...' 把含该字面量的 bash -c
        # 诊断命令当成隧道进程，ensure 脚本跳过重启 → provider.daydaymoney.com 半开挂死。
        cmd = (
            "/bin/bash -O extglob -c snap=$(command cat <&3) && "
            "pgrep -af 'autossh.*-R 8010:127.0.0.1:8010'"
        )
        self.assertFalse(self.mod.cmdline_is_real_autossh(cmd, "", "8010"))

    def test_old_loose_pattern_would_false_positive(self):
        cmd = "bash -c \"pgrep -f autossh.*-R 8010:127.0.0.1:8010\""
        self.assertRegex(cmd, r"autossh.*-R 8010:127.0.0.1:8010")
        self.assertFalse(self.mod.cmdline_is_real_autossh(cmd, "", "8010"))

    def test_probe_ok_http_status(self):
        self.assertTrue(self.mod.remote_probe_ok(curl_exit=0, http_code="200"))
        self.assertTrue(self.mod.remote_probe_ok(curl_exit=22, http_code="404"))
        self.assertTrue(self.mod.remote_probe_ok(curl_exit=52, http_code="000"))

    def test_probe_fail_timeout_or_refused(self):
        self.assertFalse(self.mod.remote_probe_ok(curl_exit=28, http_code="000"))
        self.assertFalse(self.mod.remote_probe_ok(curl_exit=7, http_code="000"))
        self.assertFalse(self.mod.remote_probe_ok(curl_exit=0, http_code="000"))


if __name__ == "__main__":
    unittest.main()
