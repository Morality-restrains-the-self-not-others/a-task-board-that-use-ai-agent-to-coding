#!/usr/bin/env python3
"""第一方 README 版权行 SSOT（OPT-20260824-013）。

20 个第一方 README.md 的版权行 `Copyright (c) <年份> <所有人>` 为手工维护，
年份/所有人漂移只能靠人工扫描发现。本门禁对第一方 README 中**出现**的
版权行强制等于 SSOT：`Copyright (c) 2025～2026 contact@daydaymoney.com`。

范围：排除 `docs/architecture/Archi/**`、`**/third_party/**`、
`trae-agent/**`、`gitService/**`、`sdk/**`（第三方/构建产物目录，版权行
不受本 SSOT 约束）。仅检查「存在版权行的 README」——未带版权行的 README
不强制补充，避免把非版权行文件误拦。

用法:
  python3 db/scripts/ci/check_copyright_ssot.py [--root <repo>]
exit 0 = 通过；exit 1 = 命中漂移版权行（列出文件:行 + 期望）。
"""
from __future__ import annotations

import argparse
import re
import sys
from pathlib import Path

ROOT = Path(__file__).resolve().parents[3]

SSOT_COPYRIGHT = "Copyright (c) 2025～2026 contact@daydaymoney.com"

COPYRIGHT_LINE_RE = re.compile(r"^\s*Copyright\b", re.IGNORECASE)

EXCLUDED_PARTS = frozenset(
    {
        "third_party",
        "trae-agent",
        "gitService",
        "sdk",
        ".git",
        "node_modules",
        ".pytest_cache",
        "Archi",
    }
)


def excluded(path: Path, root: Path) -> bool:
    """路径是否应排除（第三方 / 构建产物 / Archi 目录）。"""
    rel = path.relative_to(root)
    parts = rel.parts
    # docs/architecture/Archi/** 整目录排除
    if parts[:3] == ("docs", "architecture", "Archi"):
        return True
    return any(p in EXCLUDED_PARTS for p in parts)


def iter_readmes(root: Path) -> list[Path]:
    return [
        path
        for path in root.rglob("README.md")
        if not excluded(path, root)
    ]


def check_file(path: Path, root: Path) -> list[str]:
    rel = str(path.relative_to(root))
    try:
        text = path.read_text(encoding="utf-8", errors="replace")
    except OSError as e:
        return [f"{rel}: unreadable ({e})"]
    hits: list[str] = []
    for lineno, line in enumerate(text.splitlines(), start=1):
        if COPYRIGHT_LINE_RE.match(line) and line.strip() != SSOT_COPYRIGHT:
            hits.append(
                f"{rel}:{lineno}: copyright line {line.strip()!r} "
                f"must equal {SSOT_COPYRIGHT!r}"
            )
    return hits


def collect_violations(root: Path) -> list[str]:
    hits: list[str] = []
    for path in iter_readmes(root):
        hits.extend(check_file(path, root))
    return hits


def main(argv: list[str] | None = None) -> int:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--root", type=Path, default=ROOT)
    args = parser.parse_args(argv)
    root = args.root.resolve()
    readmes = iter_readmes(root)
    hits = collect_violations(root)
    if hits:
        print("VIOLATION (OPT-20260824-013 copyright SSOT):")
        for h in hits:
            print(f"  - {h}")
        return 1
    print(f"ok: {len(readmes)} first-party READMEs checked, copyright lines match SSOT")
    return 0


if __name__ == "__main__":
    sys.exit(main())
