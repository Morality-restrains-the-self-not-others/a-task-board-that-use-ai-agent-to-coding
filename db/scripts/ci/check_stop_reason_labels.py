#!/usr/bin/env python3
"""断言前端 serverStartHistoryDisplay.js STOP_REASON_LABELS 覆盖 shareLib/stopreason Codes()。

OPT-20260823-026：stop_reason 码表 SSOT 在 shareLib/stopreason（Go），
taskCloudService / taskEvents 均已委托该包；前端不得缺少 Go 侧已定义的码，
新增码必须同步前端，否则本门禁红。
"""
from __future__ import annotations

import re
import sys
from pathlib import Path

ROOT = Path(__file__).resolve().parents[3]

GO_FILE = ROOT / "shareLib" / "stopreason" / "stopreason.go"
FE_FILE = ROOT / "taskFE" / "app" / "src" / "utils" / "serverStartHistoryDisplay.js"


def go_codes(src: str) -> set[str]:
    m = re.search(r"func Codes\(\) \[\]string \{.*?return \[\]string\{(.*?)\}", src, re.S)
    if not m:
        raise ValueError("Codes() literal not found in stopreason.go")
    return set(re.findall(r'"([^"]+)"', m.group(1)))


def fe_keys(src: str) -> set[str]:
    m = re.search(r"const STOP_REASON_LABELS = Object\.freeze\(\{(.*?)\}\)", src, re.S)
    if not m:
        raise ValueError("STOP_REASON_LABELS not found in serverStartHistoryDisplay.js")
    return set(re.findall(r"^\s*([\w]+):", m.group(1), re.M))


def main() -> int:
    if not GO_FILE.exists():
        print(f"missing shareLib/stopreason/stopreason.go at {GO_FILE}")
        return 1
    if not FE_FILE.exists():
        print(f"missing taskFE serverStartHistoryDisplay.js at {FE_FILE}")
        return 1
    codes = go_codes(GO_FILE.read_text())
    keys = fe_keys(FE_FILE.read_text())
    missing = sorted(codes - keys)
    if missing:
        print(f"DRIFT: 前端 STOP_REASON_LABELS 缺少 stopreason.Codes() 定义的码: {missing}")
        print(f"  Go SSOT: {sorted(codes)}")
        print(f"  FE keys: {sorted(keys)}")
        print("  修复：在 taskFE/app/src/utils/serverStartHistoryDisplay.js 补齐对应中文标签")
        return 1
    print(f"OK: 前端覆盖全部 {len(codes)} 个 stop_reason 码（FE 另有 {len(keys - codes)} 个额外码）")
    return 0


if __name__ == "__main__":
    sys.exit(main())
