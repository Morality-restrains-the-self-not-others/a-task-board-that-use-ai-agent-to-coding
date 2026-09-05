#!/usr/bin/env python3
"""OPT-20260807-036 回归门禁：FE URL 尾斜杠 vs 后端 HasSuffix 精确匹配契约。

2026-08-07 交易流水页 `units` 无尾斜杠 404：FE 调用 `/api/tenant/{tid}/billing/units`
（无尾斜杠），而 taskBill catch-all 用 `strings.HasSuffix(p, "/billing/units/")`
精确匹配 → 请求落入 default 404。同类隐患可能残留其他模块。

本门禁静态扫描 taskFE app/src 的 .js/.vue/.ts 中所有含 `/api/` 的字符串字面量：

1. 从 Go 服务源码提取「尾斜杠叶子」= 正向（非 `!`）`strings.HasSuffix(<var>, "X/")`
   中 `X/` 开 `/` 收 `/` 的匹配器；若同条件含 `strings.Contains(<var>, "C")` 则记录
   前置条件 C（如 `/billing/orders/` + `/pay/`）。
2. 对每个 FE URL 模板，去掉查询串/锚点（感知 `${...}` 插值），若路径部分不以 `/`
   结尾，且去掉尾斜杠后命中某叶子（叶子匹配时满足其 Contains 前置条件）→ 阻断。

豁免（与 check_fe_url_placeholder.py 一致）：
- 注释内（行注释 / 块注释）
- `it()`/`test()`/`describe()` 第一个参数（测试描述串）
- `.replace()`/`.replaceAll()` 替换串参数

用法：pre-commit 门禁 `python3 db/scripts/ci/check_fe_url_trailing_slash.py`
exit 0 = 通过；exit 1 = 命中缺少尾斜杠的 URL（列出文件:行 + 后端匹配器）。
"""
import os
import re
import sys
from pathlib import Path

ROOT = Path(__file__).resolve().parents[3]
FE_SRC = str(ROOT / "taskFE" / "app" / "src")

EXTENSIONS = (".js", ".vue", ".ts")

# Go 服务源码目录（含 catch-all HasSuffix 路由匹配的服务）
GO_SRC_DIRS = [
    "taskBill/src",
    "taskProjectService/src",
    "taskAuth/src",
    "taskCloud/src",
    "taskTaskService/src",
    "taskTenantService/src",
    "taskGateway",
    "shareLib",
]

# 字符串字面量（单/双引号 + 反引号模板，非贪婪）
STRING_RE = re.compile(r"""(['"`])(.*?)(?<!\\)\1""", re.DOTALL)

HAS_SUFFIX_RE = re.compile(r'(?<!\!)strings\.HasSuffix\(\s*(\w+)\s*,\s*"([^"]+)"\s*\)')
CONTAINS_RE = re.compile(r'strings\.Contains\(\s*(\w+)\s*,\s*"([^"]+)"\s*\)')

TEST_FN_RE = re.compile(r"\b(?:it|test|describe|skip|only)\($")
REPLACE_RE = re.compile(r"\.replace(?:All)?\(")


def _comment_spans(src: str):
    """计算注释区间的 [(start, end)]，跳过字符串内的 // 与 /*。"""
    spans = []
    i = 0
    n = len(src)
    in_str = None
    while i < n:
        ch = src[i]
        if in_str:
            if ch == "\\":
                i += 2
                continue
            if ch == in_str:
                in_str = None
            i += 1
            continue
        if ch in "'\"`":
            in_str = ch
            i += 1
            continue
        if ch == "/" and i + 1 < n and src[i + 1] == "/":
            j = src.find("\n", i)
            spans.append((i, n if j == -1 else j))
            i = n if j == -1 else j
            continue
        if ch == "/" and i + 1 < n and src[i + 1] == "*":
            j = src.find("*/", i + 2)
            spans.append((i, n if j == -1 else j + 2))
            i = n if j == -1 else j + 2
            continue
        i += 1
    return spans


def _in_comment(spans, pos):
    for s, e in spans:
        if s <= pos < e:
            return True
    return False


def _is_test_description(before: str) -> bool:
    return bool(TEST_FN_RE.search(before.rstrip()))


def _is_replace_replacement(before: str) -> bool:
    if not REPLACE_RE.search(before):
        return False
    # 粗略判断：字符串位于 .replace()/.replaceAll() 调用参数区
    return True


def extract_backend_leaves() -> list:
    """从 Go 服务源码提取 (contains, leaf_suffix) 尾斜杠匹配器。"""
    leaves = []
    for rel in GO_SRC_DIRS:
        base = ROOT / rel
        if not base.is_dir():
            continue
        for dirpath, _dirs, files in os.walk(base):
            for name in files:
                if not name.endswith(".go") or name.endswith("_test.go"):
                    continue
                p = os.path.join(dirpath, name)
                try:
                    with open(p, encoding="utf-8") as f:
                        src = f.read()
                except OSError:
                    continue
                for line in src.splitlines():
                    for m in HAS_SUFFIX_RE.finditer(line):
                        var, lit = m.group(1), m.group(2)
                        if not (lit.startswith("/") and lit.endswith("/") and len(lit) > 1):
                            continue
                        contains = None
                        cm = CONTAINS_RE.search(line)
                        if cm and cm.group(1) == var:
                            contains = cm.group(2)
                        leaves.append((contains, lit))
    # 去重（保序）
    seen = set()
    uniq = []
    for c, s in leaves:
        key = (c, s)
        if key not in seen:
            seen.add(key)
            uniq.append(key)
    return uniq


def strip_query(path: str) -> str:
    """去掉查询串/锚点，感知 `${...}` 插值（其内部 ? 不视为查询分隔）。"""
    depth = 0
    i = 0
    n = len(path)
    while i < n:
        if path[i] == "$" and i + 1 < n and path[i + 1] == "{":
            depth += 1
            i += 2
            continue
        if depth > 0:
            if path[i] == "{":
                depth += 1
            elif path[i] == "}":
                depth -= 1
            i += 1
            continue
        if path[i] in "?#":
            return path[:i]
        i += 1
    return path


def scan_file(path: str, leaves: list):
    hits = []
    with open(path, encoding="utf-8") as f:
        src = f.read()
    comments = _comment_spans(src)
    for m in STRING_RE.finditer(src):
        start = m.start()
        content = m.group(2)
        if "/api/" not in content:
            continue
        if _in_comment(comments, start):
            continue
        before = src[max(0, start - 200):start]
        if _is_test_description(before):
            continue
        if _is_replace_replacement(before):
            continue
        path = strip_query(content)
        if path.endswith("/"):
            continue
        # 命中尾斜杠叶子（长匹配优先）
        for contains, leaf in sorted(leaves, key=lambda x: len(x[1]), reverse=True):
            if contains is not None and contains not in content:
                continue
            if path.endswith(leaf.rstrip("/")):
                hits.append((content.strip()[:120], leaf, contains))
                break
    return hits


def main():
    target = sys.argv[1] if len(sys.argv) > 1 else FE_SRC
    if not os.path.isdir(target):
        print(f"skip: {target} 不存在", file=sys.stderr)
        return 0
    leaves = extract_backend_leaves()
    if not leaves:
        print("⚠ 未提取到后端尾斜杠叶子，跳过", file=sys.stderr)
        return 0
    all_hits = []
    for dirpath, _dirs, files in os.walk(target):
        for name in sorted(files):
            if not name.endswith(EXTENSIONS):
                continue
            p = os.path.join(dirpath, name)
            rel = os.path.relpath(p, ROOT)
            for content, leaf, contains in scan_file(p, leaves):
                suffix = f"（前置 {contains}）" if contains else ""
                all_hits.append((rel, content, leaf, suffix))
    if all_hits:
        print(f"⛔ OPT-20260807-036 FE URL 缺尾斜杠 {len(all_hits)} 处：")
        for rel, content, leaf, suffix in all_hits:
            print(f"    {rel}: 后端 HasSuffix 需尾斜杠 {leaf}{suffix}，FE 缺：{content}")
        return 1
    print(f"✓ OPT-20260807-036 FE URL 尾斜杠扫描通过（{len(leaves)} 个后端叶子，0 处缺失）")
    return 0


if __name__ == "__main__":
    sys.exit(main())
