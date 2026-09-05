#!/usr/bin/env python3
"""OPT-20260811-068 回归门禁：Vue SFC 模板不得调用 <script setup> 未定义/未导入的标识符。

背景：BillingDashboard.vue 模板调用了不存在的 `centsToYuanInternalStr`，生产有交易数据
时渲染抛错导致整页空白，且 vite build 无编译期错误。本门禁静态扫描 taskFE app/src 全部
.vue 文件，对每个 SFC 提取 <template> 与 <script setup>：

1. 在模板插值 `{{ ... }}` 与指令/绑定属性值（`@click="..."`、`:prop="fn()"`、`v-if` 等）
   中找出「裸函数调用」标识符（紧跟 `(` 的标识符，排除成员调用 `obj.fn(`）。
2. 在 <script setup> 中收集已定义标识符：import、function 声明、const/let/var
   （含解构）、defineProps/defineEmits/withDefaults 的键、ref/computed 等。
3. 命中「调用但未定义」且不在全局白名单（JS/Vue 内建、浏览器 API）即阻断。

豁免（防假阳性）：
- 成员调用基名（`a.b(` 只检查是否 a 也未定义，见下；默认只拦裸调用）
- 组件标签（`<Foo>`）与指令名（`v-if`）不是函数调用
- 无 <script setup> 的 SFC（Options API / 纯展示组件）跳过——无法可靠判定定义集
- 测试目录 `**/__tests__/**`、`**/*.spec.vue`、`**/*.test.vue` 跳过
- 常见 JS/Vue 全局标识符白名单

用法：pre-commit 门禁 `python3 db/scripts/ci/check_vue_template_undefined_identifiers.py`
exit 0 = 通过；exit 1 = 命中未定义调用（列出 文件:行: 标识符）。
可选 `--src <dir>` 覆盖待扫描目录（默认 taskFE/app/src）。
"""
from __future__ import annotations

import argparse
import re
import sys
from pathlib import Path

ROOT = Path(__file__).resolve().parents[3]
DEFAULT_SRC = str(ROOT / "taskFE" / "app" / "src")

VUE_GLOB = "*.vue"

# 模板中的 JS 片段：{{ ... }} 插值
INTERP_RE = re.compile(r"\{\{([^{}]*)\}\}")

# 模板中的绑定/指令属性值：:prop="expr" @click="fn" @click.prevent="fn" v-if="expr" v-bind:x="expr"
ATTR_VALUE_RE = re.compile(
    r"""(?:[@:][A-Za-z_][\w.-]*|v-[\w.-]+)\s*=\s*(?:"([^"]*)"|'([^']*)')"""
)

# 裸函数调用：标识符紧跟 (，前导不能是 word/./$（成员调用 obj.fn( 不拦基名）
CALL_RE = re.compile(r"(?<![$\w.])([A-Za-z_$][\w$]*)\s*\(")

# <script setup> 内定义提取
IMPORT_RE = re.compile(
    r"""import\s+(?:type\s+)?(?:([A-Za-z_$][\w$]*)\s*,?\s*)?(?:([A-Za-z_$][\w$]*)\s*from)?[\s{]""",
    re.MULTILINE,
)
IMPORT_NAMED_RE = re.compile(r"import\s+(?:type\s+)?\{([^}]*)\}\s*from\s*['\"]", re.MULTILINE)
IMPORT_STAR_RE = re.compile(r"import\s+(?:type\s+)?\*\s+as\s+([A-Za-z_$][\w$]*)", re.MULTILINE)
FUNC_DECL_RE = re.compile(r"\bfunction\s+([A-Za-z_$][\w$]*)", re.MULTILINE)
CONST_DECL_RE = re.compile(
    r"\b(?:const|let|var)\s+(?:type\s+)?([A-Za-z_$][\w$]*)\s*=", re.MULTILINE
)
DESTRUCT_OBJ_RE = re.compile(r"\b(?:const|let|var)\s*\{([^}]*)\}\s*=", re.MULTILINE)
DESTRUCT_ARR_RE = re.compile(r"\b(?:const|let|var)\s*\[([^\]]*)\]\s*=", re.MULTILINE)

OBJ_KEY_RE = re.compile(r"([A-Za-z_$][\w$]*)\s*:")
ARR_ITEM_RE = re.compile(r"['\"]?([A-Za-z_$][\w$]*)['\"]?")

DEFINE_FN_RE = re.compile(r"\b(?:defineProps|defineEmits|withDefaults)\s*\(")

# 跳过这些 JS 控制关键字（模板表达式里出现 if( 等一般是无害/罕见，防御性跳过）
SKIP_KEYWORDS = frozenset({
    "if", "for", "while", "switch", "catch", "function", "return", "typeof",
    "instanceof", "new", "void", "delete", "do", "else", "case", "default",
    "throw", "in", "of", "with", "this", "super", "class", "extends", "yield",
    "await", "async", "import", "export", "try", "finally",
})

# JS/Vue 内建 + 浏览器 API 白名单
GLOBALS = frozenset({
    # JS 内建
    "Math", "JSON", "Date", "Number", "String", "Boolean", "Array", "Object",
    "Symbol", "RegExp", "Error", "TypeError", "RangeError", "Promise", "Set",
    "Map", "WeakSet", "WeakMap", "BigInt", "Infinity", "NaN", "undefined",
    "parseInt", "parseFloat", "isNaN", "isFinite", "encodeURI",
    "encodeURIComponent", "decodeURI", "decodeURIComponent", "Intl", "globalThis",
    # 浏览器 / DOM
    "console", "window", "document", "localStorage", "sessionStorage",
    "navigator", "URL", "URLSearchParams", "fetch", "FormData", "Blob", "File",
    "FileReader", "Image", "Audio", "setTimeout", "setInterval", "clearTimeout",
    "clearInterval", "requestAnimationFrame", "cancelAnimationFrame", "crypto",
    "atob", "btoa", "Element", "Node", "Event", "CustomEvent", "XMLHttpRequest",
    "structuredClone", "history", "location", "screen", "performance", "IntersectionObserver",
    "ResizeObserver", "MutationObserver", "AbortController", "AbortSignal",
    "Intl", "structuredClone", "requestIdleCallback", "cancelIdleCallback",
    # Vue 模板内建（$ 前缀）
    "$t", "$tc", "$d", "$n", "$emit", "$refs", "$props", "$attrs", "$slots",
    "$scopedSlots", "$root", "$parent", "$children", "$options", "$el",
    "$nextTick", "$forceUpdate", "$set", "$delete", "$watch", "$on", "$once",
    "$off", "$mount", "$destroy",
})

# 已确认的存量真实缺陷（非假阳性）。这些会随着对应 OPT 修复后从本表移除。
# 键：(相对 src 路径, 标识符)。禁止静默扩容——每个新增项必须附带修复追踪编号。
KNOWN_ISSUES: dict[tuple[str, str], str] = {}


def extract_block(text: str, open_re: re.Pattern, close_tag: str) -> tuple[str, int, int] | None:
    """提取匹配开标签后的配对文本（计数嵌套同名开标签），返回 (inner, start_line_1based, block_start_idx)。"""
    m = open_re.search(text)
    if not m:
        return None
    start = m.end()
    open_tag = m.group(0)
    depth = 1
    i = start
    open_name_re = re.compile(r"<template\b", re.IGNORECASE)
    close_name_re = re.compile(close_tag, re.IGNORECASE)
    while i < len(text):
        nxt_open = open_name_re.search(text, i)
        nxt_close = close_name_re.search(text, i)
        if nxt_close is None:
            break
        if nxt_open is not None and nxt_open.start() < nxt_close.start():
            depth += 1
            i = nxt_open.end()
        else:
            depth -= 1
            if depth == 0:
                line = text.count("\n", 0, m.start()) + 1
                return text[start:nxt_close.start()], line, start
            i = nxt_close.end()
    return None


def _match_balanced(text: str, open_idx: int) -> tuple[str, int] | None:
    """从 open_idx 处的开括号开始，做括号/方括号/花括号/字符串感知的配对扫描。

    返回 (inner, close_idx)，inner 为括号内文本。遇到不匹配返回 None。
    """
    stack: list[str] = []
    i = open_idx
    n = len(text)
    pairs = {")": "(", "]": "[", "}": "{"}
    quotes = {"'", '"', "`"}
    while i < n:
        ch = text[i]
        if ch in quotes:
            q = ch
            j = i + 1
            while j < n:
                if text[j] == "\\":
                    j += 2
                    continue
                if text[j] == q:
                    break
                j += 1
            i = j + 1
            continue
        if ch in "([{":
            stack.append(ch)
        elif ch in ")]}":
            if not stack or stack[-1] != pairs[ch]:
                return None
            stack.pop()
            if not stack:
                return text[open_idx + 1:i], i
        i += 1
    return None


def _obj_keys(body: str) -> set[str]:
    keys: set[str] = set()
    for m in OBJ_KEY_RE.finditer(body):
        keys.add(m.group(1))
    return keys


def _arr_items(body: str) -> set[str]:
    items: set[str] = set()
    for m in ARR_ITEM_RE.finditer(body):
        items.add(m.group(1))
    return items


def collect_script_defs(script: str) -> set[str]:
    """从 <script setup> 文本收集已定义标识符集合。"""
    defs: set[str] = set()

    for m in IMPORT_STAR_RE.finditer(script):
        defs.add(m.group(1))
    # 默认导入：import X from '...' 或 import X, { ... } from '...'
    for m in re.finditer(r"import\s+(?:type\s+)?([A-Za-z_$][\w$]*)\s*(?:,\s*\{[^}]*\})?\s*from\s*['\"]", script):
        defs.add(m.group(1))
    # 命名导入：import { A, B as C } from '...'
    for m in IMPORT_NAMED_RE.finditer(script):
        for token in m.group(1).split(","):
            token = token.strip()
            if not token:
                continue
            token = re.sub(r"\btype\s+", "", token).strip()
            if " as " in token:
                token = token.split(" as ")[-1].strip()
            token = re.sub(r"^type\s+", "", token).strip()
            if token:
                defs.add(token)

    defs.update(FUNC_DECL_RE.findall(script))
    defs.update(CONST_DECL_RE.findall(script))
    for m in DESTRUCT_OBJ_RE.finditer(script):
        for key in OBJ_KEY_RE.finditer(m.group(1)):
            defs.add(key.group(1))
        for item in ARR_ITEM_RE.finditer(m.group(1)):
            defs.add(item.group(1))
    for m in DESTRUCT_ARR_RE.finditer(script):
        for item in ARR_ITEM_RE.finditer(m.group(1)):
            defs.add(item.group(1))

    # defineProps / defineEmits / withDefaults 的键（对象或数组），支持嵌套类型对象
    for m in DEFINE_FN_RE.finditer(script):
        open_idx = script.find("(", m.start())
        if open_idx < 0:
            continue
        bal = _match_balanced(script, open_idx)
        if not bal:
            continue
        body = bal[0].strip()
        if body.startswith("{"):
            defs |= _obj_keys(body)
        elif body.startswith("["):
            defs |= _arr_items(body)

    return defs


def template_call_sites(template: str, base_line: int) -> list[tuple[str, int]]:
    """返回模板中发现的裸调用 (identifier, line)。"""
    found: list[tuple[str, int]] = []
    # 先定位每个插值 / 属性值片段的行号，再在片段内找调用
    for pat in (INTERP_RE, ATTR_VALUE_RE):
        for m in pat.finditer(template):
            line = template.count("\n", 0, m.start()) + base_line
            snippet = m.group(1) or m.group(2) or m.group(3)
            if not snippet:
                continue
            for cm in CALL_RE.finditer(snippet):
                ident = cm.group(1)
                if ident in SKIP_KEYWORDS:
                    continue
                found.append((ident, line))
    return found


def analyze_sfc(path: Path, rel: str | None = None) -> list[tuple[str, int]]:
    text = path.read_text(encoding="utf-8", errors="replace")
    findings: list[tuple[str, int]] = []

    tmpl = extract_block(text, re.compile(r"<template\b"), r"</template\s*>")
    if not tmpl:
        return findings
    tpl_inner, tpl_line, _ = tmpl

    script = extract_block(text, re.compile(r"<script\s+setup\b"), r"</script\s*>")
    if not script:
        # 无 <script setup>：Options API 无法可靠判定，跳过
        return findings
    scr_inner, _, _ = script

    defs = collect_script_defs(scr_inner)
    defs |= GLOBALS

    rel = rel or str(path.name)
    for ident, line in template_call_sites(tpl_inner, tpl_line):
        if ident in defs:
            continue
        reason = KNOWN_ISSUES.get((rel, ident))
        if reason:
            print(f"忽略存量已知缺陷 {rel}:{line}: {ident}(...) — {reason}", file=sys.stderr)
            continue
        findings.append((ident, line))
    return findings


def scan(src: str) -> list[tuple[str, Path, int]]:
    root = Path(src)
    findings: list[tuple[str, Path, int]] = []
    if not root.exists():
        print(f"错误: 扫描目录不存在 {root}", file=sys.stderr)
        return [("__dir_missing__", root, 0)]
    for p in sorted(root.rglob(VUE_GLOB)):
        rel = str(p.relative_to(root))
        if "/__tests__/" in rel or rel.endswith(".spec.vue") or rel.endswith(".test.vue"):
            continue
        for ident, line in analyze_sfc(p, rel=rel):
            findings.append((ident, p, line))
    return findings


def main() -> int:
    ap = argparse.ArgumentParser(description=__doc__)
    ap.add_argument("--src", default=DEFAULT_SRC)
    args = ap.parse_args()

    findings = scan(args.src)
    if findings:
        for ident, path, line in findings:
            if ident == "__dir_missing__":
                continue
            print(f"{path}:{line}: 模板调用未定义标识符 {ident}(...)")
        if any(ident == "__dir_missing__" for ident, _, _ in findings):
            return 2
        return 1
    return 0


if __name__ == "__main__":
    sys.exit(main())
