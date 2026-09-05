#!/usr/bin/env python3
"""微信支付必须使用仓内 sdk/wechatpay-go（元规则 54 / ADR-0032）。

依赖 github.com/wechatpay-apiv3/wechatpay-go 的服务必须 replace 到
仓库根 sdk/wechatpay-go；禁止社区微信支付 SDK。

用法:
  python3 db/scripts/ci/check_wechatpay_go_sdk.py
"""
from __future__ import annotations

import argparse
import re
import sys
from pathlib import Path

ROOT = Path(__file__).resolve().parents[3]

WECHATPAY_MODULE = "github.com/wechatpay-apiv3/wechatpay-go"
SDK_DIRNAME = "sdk/wechatpay-go"

SKIP_DIR_PARTS = frozenset(
    {
        "third_party",
        "vendor",
        "node_modules",
        "gitlab-ce",
        "gitlab_home",
        ".git",
        "sdk",
    }
)

FORBIDDEN_IMPORT_PREFIXES = (
    "github.com/go-pay/gopay",
    "github.com/objcoding/wxpay",
    "github.com/silenceper/wechat",
    "github.com/medivhzhan/weapp",
    "github.com/wechatpay/wechatpay-go",
)

META_FILES = (
    ".ai/01_project_constraints/59_wechatpay_go_sdk.md",
    ".cursor/rules/wechatpay-go-sdk.mdc",
    "docs/adr/0032-wechatpay-go-sdk.md",
)

REPLACE_RE = re.compile(
    r"replace\s+"
    + re.escape(WECHATPAY_MODULE)
    + r"\s+=>\s+(\S+)"
    r"|"
    r"^\s+"
    + re.escape(WECHATPAY_MODULE)
    + r"\s+=>\s+(\S+)",
    re.MULTILINE,
)
IMPORT_PATH_RE = re.compile(
    r'^\s*(?:[\w.]+)?\s*"(github\.com/[^"]+)"',
    re.MULTILINE,
)
COMMENT_LINE_RE = re.compile(r"^\s*//")


def strip_go_mod_comments(text: str) -> str:
    lines = []
    for line in text.splitlines():
        stripped = line.split("//", 1)[0]
        lines.append(stripped)
    return "\n".join(lines)


def go_mod_requires_wechatpay(text: str) -> bool:
    return WECHATPAY_MODULE in strip_go_mod_comments(text)


def go_mod_replace_target(text: str) -> str | None:
    cleaned = strip_go_mod_comments(text)
    match = REPLACE_RE.search(cleaned)
    if not match:
        return None
    return (match.group(1) or match.group(2) or "").strip()


def replace_resolves_to_sdk(go_mod_path: Path, target: str, root: Path) -> bool:
    if not target:
        return False
    resolved = (go_mod_path.parent / target).resolve()
    expected = (root / SDK_DIRNAME).resolve()
    return resolved == expected


def iter_go_mod_files(root: Path) -> list[Path]:
    out: list[Path] = []
    for path in root.rglob("go.mod"):
        parts = path.relative_to(root).parts
        if any(p in SKIP_DIR_PARTS for p in parts[:-1]):
            continue
        if path.parent.name.endswith("-wt"):
            continue
        out.append(path)
    return out


def iter_go_files(root: Path) -> list[Path]:
    out: list[Path] = []
    for path in root.rglob("*.go"):
        parts = path.relative_to(root).parts
        if any(p in SKIP_DIR_PARTS for p in parts):
            continue
        out.append(path)
    return out


def check_go_mod(path: Path, root: Path) -> list[str]:
    rel = str(path.relative_to(root))
    text = path.read_text(encoding="utf-8")
    if not go_mod_requires_wechatpay(text):
        return []
    target = go_mod_replace_target(text)
    if not target:
        return [
            (
                f"{rel}: requires {WECHATPAY_MODULE} but missing "
                f"replace => {SDK_DIRNAME}"
            )
        ]
    if not replace_resolves_to_sdk(path, target, root):
        return [
            (
                f"{rel}: replace {WECHATPAY_MODULE} => {target} "
                f"must resolve to {SDK_DIRNAME}"
            )
        ]
    return []


def check_go_imports(path: Path, root: Path) -> list[str]:
    rel = str(path.relative_to(root))
    text = path.read_text(encoding="utf-8", errors="replace")
    hits = []
    for match in IMPORT_PATH_RE.finditer(text):
        line_start = text.rfind("\n", 0, match.start()) + 1
        line = text[line_start : match.start()]
        if COMMENT_LINE_RE.match(line):
            continue
        import_path = match.group(1)
        for prefix in FORBIDDEN_IMPORT_PREFIXES:
            if import_path == prefix or import_path.startswith(prefix + "/"):
                hits.append(f"{rel}: forbidden WeChat Pay SDK import {import_path}")
    return hits


def collect_violations(root: Path, check_meta: bool = True) -> list[str]:
    hits: list[str] = []
    sdk_dir = root / SDK_DIRNAME
    if not sdk_dir.is_dir():
        hits.append(f"{SDK_DIRNAME}: missing vendored WeChat Pay Go SDK directory")
    else:
        sdk_mod = sdk_dir / "go.mod"
        if not sdk_mod.is_file():
            hits.append(f"{SDK_DIRNAME}/go.mod: missing")
        else:
            first = sdk_mod.read_text(encoding="utf-8").splitlines()[:4]
            if not any(line.strip() == f"module {WECHATPAY_MODULE}" for line in first):
                hits.append(
                    f"{SDK_DIRNAME}/go.mod: module must be {WECHATPAY_MODULE}"
                )

    for go_mod in iter_go_mod_files(root):
        hits.extend(check_go_mod(go_mod, root))

    for go_file in iter_go_files(root):
        hits.extend(check_go_imports(go_file, root))

    if check_meta:
        for rel in META_FILES:
            if not (root / rel).is_file():
                hits.append(f"{rel}: missing meta file")
    return hits


def main(argv: list[str] | None = None) -> int:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--root", type=Path, default=ROOT)
    args = parser.parse_args(argv)
    root = args.root.resolve()
    hits = collect_violations(root, check_meta=True)
    if hits:
        print("VIOLATION (rule 59_wechatpay_go_sdk.md / ADR-0032):")
        for h in hits:
            print(f"  - {h}")
        return 1
    print("ok: WeChat Pay uses sdk/wechatpay-go")
    return 0


if __name__ == "__main__":
    sys.exit(main())
