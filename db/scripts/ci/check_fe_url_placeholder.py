#!/usr/bin/env python3
"""OPT-20260807-032 回归门禁：任务源 URL 模板禁止迁移占位符残留。

e105bad API 路径迁移先后产生 ~30 处（7bd031b 修）与 5 处字面量 `$1`
（2026-08-07 修）URL 损坏，以及 DeliverableSystemList.vue 硬编码 `${1}`
（OPT-20260807-053，2026-08-08 修）。纯人工迁移易漏，本门禁静态扫描 taskFE
app/src 的 .js/.vue/.ts 源中所有含 `/api/` 的字符串字面量，命中以下残留即阻断：

- 字面量 `$1`（regex 反向引用/替换残留；若位于 `.replace()`/`.replaceAll()`
  的替换串参数内则豁免——那是合法捕获组引用）
- 字面量 `{sub}`/`{id}`/`{tenant_id}`/`{tid}`/`{wsId}`/`{workspaceId}`
  （应为 `${...}` 模板插值或真实路径段；前置 `$` 的 `${...}` 插值豁免）

豁免：
- 注释内（行注释 / 块注释 / JSDoc）的路径说明文字
- `it()`/`test()`/`describe()` 第一个参数（测试描述串，非真实请求目标）
- `.replace()`/`.replaceAll()` 替换串参数（`$1` 为合法捕获组引用）

用法：pre-commit 门禁 `python3 db/scripts/ci/check_fe_url_placeholder.py`
exit 0 = 通过；exit 1 = 命中残留（列出文件:行）。
"""
import os
import re
import sys
from pathlib import Path

ROOT = Path(__file__).resolve().parents[3]
FE_SRC = str(ROOT / "taskFE" / "app" / "src")

EXTENSIONS = (".js", ".vue", ".ts")

LITERAL_DOLLAR = re.compile(r"\$1(?![\dA-Za-z_])")
LITERAL_BRACE = re.compile(r"(?<!\$)\{(?:sub|id|tenant_id|tid|wsId|workspaceId|path|wid)\}")

# 字符串字面量（单/双引号 + 反引号模板，非贪婪）
STRING_RE = re.compile(r"""(['"`])(.*?)(?<!\\)\1""", re.DOTALL)

# 白名单调用点
REPLACE_RE = re.compile(r"\.replace(?:All)?\(")
TEST_FN_RE = re.compile(r"\b(?:it|test|describe|skip|only)\($")


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


def _is_replace_replacement(before: str) -> bool:
    """字符串是否为 .replace()/.replaceAll() 的替换串参数（$1 豁免）。"""
    m = list(REPLACE_RE.finditer(before))
    if not m:
        return False
    call = m[-1]
    tail = before[call.end():]
    depth = 0
    in_str = None
    for i, ch in enumerate(tail):
        if in_str:
            if ch == in_str and (i == 0 or tail[i - 1] != "\\"):
                in_str = None
            continue
        if ch in "'\"":
            in_str = ch
        elif ch == "(":
            depth += 1
        elif ch == ")":
            depth -= 1
            if depth < 0:
                return False
        elif ch == "," and depth <= 1:
            return True
    return False


def _is_test_description(before: str) -> bool:
    """字符串是否为 it()/test()/describe() 的第一个参数（描述串豁免）。"""
    stripped = before.rstrip()
    m = TEST_FN_RE.search(stripped)
    if not m:
        return False
    # 测试函数名后紧跟 '（'，且字符串紧随其后（允许空白）——描述串即第一个参数
    return True


def scan_file(path: str):
    hits = []
    with open(path, encoding="utf-8") as f:
        src = f.read()
    comments = _comment_spans(src)
    for m in STRING_RE.finditer(src):
        start = m.start()
        content = m.group(2)
        if "/api/" not in content and not content.startswith("/"):
            continue
        if _in_comment(comments, start):
            continue
        before = src[max(0, start - 200):start]
        if _is_test_description(before):
            continue
        is_replacement = _is_replace_replacement(before)
        if LITERAL_DOLLAR.search(content) and not is_replacement:
            hits.append((content, "字面量 `$1`（迁移占位符残留）"))
        if LITERAL_BRACE.search(content):
            hits.append((content, "字面量 `{...}` 占位符残留（应为 ${...} 插值或真实路径段）"))
    return hits


def main():
    target = sys.argv[1] if len(sys.argv) > 1 else FE_SRC
    if not os.path.isdir(target):
        print(f"skip: {target} 不存在", file=sys.stderr)
        return 0
    all_hits = []
    for dirpath, _dirs, files in os.walk(target):
        for name in sorted(files):
            if not name.endswith(EXTENSIONS):
                continue
            p = os.path.join(dirpath, name)
            rel = os.path.relpath(p, ROOT)
            for content, why in scan_file(p):
                all_hits.append((rel, content.strip(), why))
    if all_hits:
        print(f"⛔ OPT-20260807-032 URL 占位符残留 {len(all_hits)} 处：")
        for rel, content, why in all_hits:
            print(f"    {rel}: {why}: {content[:120]}")
        return 1
    print("✓ OPT-20260807-032 URL 占位符扫描通过（无字面量 $1/{sub}/{id} 残留）")
    return 0


if __name__ == "__main__":
    sys.exit(main())
