#!/usr/bin/env python3
"""taskFE 相对导入可解析性门禁（OPT-20260811-089）。

v75 公网整站挂掉根因：提交遗漏 `saveSubjectResourceAccess.js` / `resourceGrantEffects.js`，
vite build 失败 → 空 dist → 公网 / → 302 → 404。本门禁在提交前静态校验
`taskFE/app/src/**` 中所有相对导入（./x 或 ../y）都能解析到存在的文件，
不跑完整 vite build，几秒内拦截「引用缺失模块」这类回归。

跳过：
  - 非相对导入（npm 包名、@/ 别名 — 由 vite resolve.alias 处理）
  - 导入路径带查询/哈希（如 ?raw、#）仅解析路径段

用法：
  python3 scripts/ci/check_fe_relative_imports.py            # 扫描 taskFE/app/src
  python3 scripts/ci/check_fe_relative_imports.py --src <dir> # 自定义源码目录

退出码 0 = 通过；1 = 存在无法解析的相对导入。
"""

from __future__ import annotations

import argparse
import re
import sys
from pathlib import Path

_IMPORT_RE = re.compile(
    r"""(?:import|export)\s+(?:[^'"]*?\s+from\s+)?['"]([^'"]+)['"]"""
    r"""|import\s*\(\s*['"]([^'"]+)['"]\s*\)""",
)
_ALIAS_PREFIXES = ("@/", "~", "@")

_EXT_VARIANTS = ("", ".js", ".vue", ".jsx", ".ts", ".tsx", "/index.js", "/index.vue")


def taskfe_src_root() -> Path:
    """向上搜索持有 taskFE/app/src 的 meta 根；直接运行于 taskFE 仓时用 cwd/src。"""
    d = Path(__file__).resolve().parent
    while d != d.parent:
        candidate = d / "taskFE" / "app" / "src"
        if candidate.is_dir():
            return candidate
        d = d.parent
    # 直接运行于 taskFE 仓内（src 在 cwd 下）
    cwd = Path.cwd()
    if (cwd / "src").is_dir() and (cwd / "package.json").is_file():
        return cwd / "src"
    return Path.cwd() / "taskFE" / "app" / "src"


def _resolve(base_dir: Path, spec: str) -> bool:
    """按 vite 常见的相对解析规则判断 spec 是否可解析。"""
    target = (base_dir / spec).resolve()
    for variant in _EXT_VARIANTS:
        if (target.parent / (target.name + variant)).is_file():
            return True
    # 目录默认入口（index.*）
    if target.is_dir():
        for index in ("index.js", "index.vue", "index.jsx", "index.ts"):
            if (target / index).is_file():
                return True
    return False


def scan_dir(src: Path) -> list[tuple[str, int, str]]:
    findings: list[tuple[str, int, str]] = []
    for path in sorted(src.rglob("*")):
        if not path.is_file() or path.suffix not in (".js", ".vue", ".jsx", ".ts", ".tsx"):
            continue
        text = path.read_text(encoding="utf-8", errors="replace")
        for m in _IMPORT_RE.finditer(text):
            spec = m.group(1) or m.group(2)
            if not spec:
                continue
            if not (spec.startswith("./") or spec.startswith("../")):
                continue
            # 去掉 ?raw / #fragment 等查询后缀再解析
            clean_spec = spec.split("?", 1)[0].split("#", 1)[0]
            if not clean_spec:
                continue
            if not _resolve(path.parent, clean_spec):
                findings.append((str(path.relative_to(src.parent.parent)), path.name, spec))
    return findings


def main() -> int:
    p = argparse.ArgumentParser(description="taskFE 相对导入可解析性门禁")
    p.add_argument("--src", help="taskFE/app/src 目录（默认自动探测）")
    args = p.parse_args()

    src = Path(args.src).resolve() if args.src else taskfe_src_root()
    if not src.is_dir():
        print(f"错误: src 目录不存在 {src}", file=sys.stderr)
        return 2

    findings = scan_dir(src)
    if not findings:
        print(f"OK {len(list(src.rglob('*'))):d} files scanned: all relative imports resolve")
        return 0

    for rel, fname, spec in findings:
        print(f"UNRESOLVED {rel}: {spec}", file=sys.stderr)
    print(f"{len(findings)} unresolved relative import(s) in taskFE/app/src", file=sys.stderr)
    return 1


if __name__ == "__main__":
    raise SystemExit(main())
