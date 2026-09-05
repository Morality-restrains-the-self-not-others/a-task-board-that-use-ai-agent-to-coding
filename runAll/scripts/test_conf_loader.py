"""Regression tests for conf_loader.py — OPT-20260806-057 收尾回归.

Bug: resolve_app_config_dir 的 docstring 闭合引号被截断（单个双引号而非
三连双引号），
导致 conf_loader.py SyntaxError → conf-read.py snapshot-json 崩溃 →
vite.config.js 回退默认端口 3000 → taskFE preview 与 ai-monitor(Grafana)
端口冲突启动失败。

修复前：py_compile 抛 SyntaxError（Red）；修复后：编译通过且
snapshot-json 正确输出 vue.port=4000（Green）。
"""
import importlib.util
import json
import py_compile
import subprocess
import sys
import unittest
from pathlib import Path

SCRIPTS_DIR = Path(__file__).resolve().parent


def load_module(name: str, path: Path):
    spec = importlib.util.spec_from_file_location(name, path)
    mod = importlib.util.module_from_spec(spec)
    assert spec.loader is not None
    spec.loader.exec_module(mod)
    return mod


class TestConfLoaderSyntax(unittest.TestCase):
    def test_module_compiles(self):
        """conf_loader.py 必须可编译（修复前 docstring 未闭合 SyntaxError）。"""
        py_compile.compile(str(SCRIPTS_DIR / "conf_loader.py"), doraise=True)

    def test_snapshot_vue_port_4000(self):
        """snapshot-json 必须正常输出 vue.port=4000（vite preview 端口真源）。"""
        conf_loader = load_module("conf_loader", SCRIPTS_DIR / "conf_loader.py")
        snapshot = conf_loader.build_runtime_snapshot()
        vue = snapshot.get("vue") or {}
        self.assertEqual(vue.get("port"), 4000)

    def test_snapshot_json_cli(self):
        """conf-read.py snapshot-json CLI 正常退出且输出 JSON。"""
        proc = subprocess.run(
            [sys.executable, str(SCRIPTS_DIR / "conf-read.py"), "snapshot-json"],
            capture_output=True,
            text=True,
            cwd=SCRIPTS_DIR,
        )
        self.assertEqual(proc.returncode, 0, proc.stderr)
        payload = json.loads(proc.stdout)
        self.assertEqual(payload["vue"]["port"], 4000)

    def test_taskSSE_alias_resolves_gateway_dir(self):
        conf_loader = load_module("conf_loader", SCRIPTS_DIR / "conf_loader.py")
        self.assertEqual(conf_loader.resolve_app_config_dir("taskSSE"), "gateway/task-sse")
        self.assertEqual(conf_loader.resolve_app_config_dir("task-sse"), "gateway/task-sse")
        block = conf_loader.load_app_config("taskSSE")
        self.assertTrue(block.get("port") or block.get("host"), block)


if __name__ == "__main__":
    unittest.main(verbosity=2)
