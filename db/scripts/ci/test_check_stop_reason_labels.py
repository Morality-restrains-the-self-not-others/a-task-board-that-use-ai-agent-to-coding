#!/usr/bin/env python3
"""自测 check_stop_reason_labels.py 的解析与断言逻辑（无外部依赖）。"""
from __future__ import annotations

import sys
import unittest

sys.path.insert(0, __file__.rsplit("/", 1)[0])
import check_stop_reason_labels as mod  # noqa: E402


class TestParsers(unittest.TestCase):
    GO_SRC = '''package stopreason

func Codes() []string {
\treturn []string{
\t\t"user_stop",
\t\t"stop_vm",
\t\t"instruction_idle",
\t\t"task_status_cancelled",
\t}
}
'''

    FE_SRC = '''const STOP_REASON_LABELS = Object.freeze({
  relay_stop: '本地中继停止',
  stop_vm: '用户停止虚拟机',
  user_stop: '用户点击停止服务器',
  instruction_idle: '容器指令空闲超时回收',
  task_status_cancelled: '任务进度变为已取消',
})
'''

    def test_go_codes(self):
        self.assertEqual(
            mod.go_codes(self.GO_SRC),
            {"user_stop", "stop_vm", "instruction_idle", "task_status_cancelled"},
        )

    def test_fe_keys(self):
        self.assertEqual(
            mod.fe_keys(self.FE_SRC),
            {"relay_stop", "stop_vm", "user_stop", "instruction_idle", "task_status_cancelled"},
        )

    def test_missing_key_detected(self):
        fe = self.FE_SRC.replace("task_status_cancelled:", "task_status_cancelled_dropped:")
        codes = mod.go_codes(self.GO_SRC)
        keys = mod.fe_keys(fe)
        self.assertIn("task_status_cancelled", codes - keys)

    def test_live_files_pass(self):
        # 对仓库真实文件跑一遍：前端必须覆盖 Go SSOT 全部码
        if mod.GO_FILE.exists() and mod.FE_FILE.exists():
            codes = mod.go_codes(mod.GO_FILE.read_text())
            keys = mod.fe_keys(mod.FE_FILE.read_text())
            self.assertTrue(codes <= keys, f"missing: {sorted(codes - keys)}")
        else:
            self.skipTest("live files missing in this checkout")


if __name__ == "__main__":
    unittest.main()
