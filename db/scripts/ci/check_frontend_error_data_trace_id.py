#!/usr/bin/env python3
"""Optional scan: Vue templates with text-red-600/text-danger error nodes missing data-traceId nearby.

Heuristic only — exits 0 by default unless --strict.
"""
from __future__ import annotations
import argparse
import re
import sys
from pathlib import Path

ROOT = Path(__file__).resolve().parents[3]
# 前端源码路径（taskFE 是当前主前端项目；task2app/front_project 为历史遗留，已移除）
_FRONT_CANDIDATES = [
    ROOT / "taskFE/app/src",
    ROOT / "task2app/front_project/app/src",
]
FRONT_DIRS = [d for d in _FRONT_CANDIDATES if d.is_dir()]
CLASS_RE = re.compile(r'class="[^"]*(?:text-red-[5-8]00|text-danger|bg-red-50)[^"]*"')
TRACE_RE = re.compile(r"data-traceId")


def main() -> int:
    ap = argparse.ArgumentParser()
    ap.add_argument("--strict", action="store_true")
    args = ap.parse_args()
    suspects = []
    for front_dir in FRONT_DIRS:
        for p in front_dir.rglob("*.vue"):
            text = p.read_text(encoding="utf-8", errors="replace")
            for m in CLASS_RE.finditer(text):
                window = text[max(0, m.start() - 200) : m.end() + 200]
                if "error" not in window.lower() and "失败" not in window and "err" not in window.lower():
                    continue
                if TRACE_RE.search(window):
                    continue
                suspects.append(f"{p.relative_to(ROOT)}:{text.count(chr(10), 0, m.start())+1}")
    print(f"suspects without nearby data-traceId: {len(suspects)}")
    for s in suspects[:50]:
        print(" ", s)
    if args.strict and suspects:
        return 1
    return 0


if __name__ == "__main__":
    sys.exit(main())
