#!/usr/bin/env python3
"""NFR 文档「幂等性审视」表门禁。

元规则 48 / 5-nfr Hard Gate 要求 NFR 澄清文档含 `## 幂等性审视` 表；本脚本
扫描 docs/superpowers/plans/*-nfr-clarification.md，防止评审遗漏。

模式：
- 默认（warn）: 存量文件缺表仅打印 WARN，不阻断（存量 200+ 文件大多缺表）。
- --strict [--files <glob,>] : 对指定（新/改）文件缺表以非零退出阻断；无 --files
  时对所有缺表文件阻断。

用法:
  check_nfr_idempotency_table.py                 # warn 全量
  check_nfr_idempotency_table.py --strict --files 'docs/superpowers/plans/2026-08-*.md'
"""
from __future__ import annotations

import argparse
import re
import sys
from pathlib import Path

ROOT = Path(__file__).resolve().parents[3]
PLANS = ROOT / "docs" / "superpowers" / "plans"
SECTION_RE = re.compile(r"^##\s*幂等性审视\s*$", re.MULTILINE)


def scan() -> list[Path]:
    """返回缺「幂等性审视」节的 NFR 澄清文档。"""
    missing: list[Path] = []
    if not PLANS.is_dir():
        return missing
    for p in sorted(PLANS.glob("*nfr*clarification.md")):
        text = p.read_text(encoding="utf-8", errors="replace")
        if not SECTION_RE.search(text):
            missing.append(p)
    return missing


def main(argv: list[str] | None = None) -> int:
    ap = argparse.ArgumentParser(description=__doc__)
    ap.add_argument("--strict", action="store_true", help="缺失即以非零退出阻断")
    ap.add_argument("--files", default="", help="逗号分隔的 glob（仅检查这些文件）")
    args = ap.parse_args(argv)

    if args.files:
        targets: list[Path] = []
        for pattern in args.files.split(","):
            pattern = pattern.strip()
            if not pattern:
                continue
            hits = sorted(ROOT.glob(pattern))
            targets.extend(hits)
        seen: set[Path] = set()
        files = [p for p in targets if not (p in seen or seen.add(p))]
    else:
        files = scan()

    missing = [p for p in files if not SECTION_RE.search(p.read_text(encoding="utf-8", errors="replace"))]
    for p in missing:
        rel = p.relative_to(ROOT)
        if args.strict:
            print(f"ERROR {rel}: 缺少「## 幂等性审视」表（硬约束 48 / 5-nfr）")
        else:
            print(f"WARN  {rel}: 缺少「## 幂等性审视」表（存量容忍，新文档应补）")
    if missing:
        print(f"nfr-idempotency-table: {len(missing)} file(s) missing section")
    else:
        print(f"ok: {len(files)} NFR file(s) contain 幂等性审视")
    return 1 if args.strict and missing else 0


if __name__ == "__main__":
    sys.exit(main())
