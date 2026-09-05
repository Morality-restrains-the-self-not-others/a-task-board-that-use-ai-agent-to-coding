#!/usr/bin/env python3
"""手动部署文档脚本/配置路径防漂移门禁（OPT-20260824-018）。

`docs/runbooks/manual-full-deployment.md` 是事故恢复与人工部署的唯一依据，
引用约 50 个脚本/配置文件路径（build.sh / run.sh / health.sh / migrate.sh 等）。
这些路径为手工维护，脚本改名/移动会产生文档漂移，直接导致手动部署失败。

本门禁解析文档中代码块里的 `bash <path>` / `./<path>` 引用，按 `cd <dir>`
上下文解析后 stat 存在性校验：

- 仅在 ``` 代码块内解析；每次进入新代码块重置 `cd` 上下文
- 解析 `cd <literal-dir> && bash …` 与 `cd <literal-dir>` 后接命令两种写法
- `cd <service-dir>` 等占位符视为不可解析，置空上下文
- 跳过含 `<`/`>` 占位符、`bash -c` 内联片段、无法解析的裸命令名

用法:
  python3 db/scripts/ci/check_deployment_doc_paths.py [--root <repo>] [--doc <rel-path>]
exit 0 = 通过；exit 1 = 命中不存在的路径（列出 文件:行 + 解析路径）。
"""
from __future__ import annotations

import argparse
import re
import sys
from pathlib import Path

ROOT = Path(__file__).resolve().parents[3]

DEFAULT_DOC = "docs/runbooks/manual-full-deployment.md"

CD_RE = re.compile(r"^\s*cd\s+([^\s&;|]+)")
BASH_RE = re.compile(r"\bbash\s+(\S+)")
DOTSLASH_RE = re.compile(r"(?<![\w./])\./(\S+)")
TOKEN_RE = re.compile(r"[./\w_-]+")


def parse_candidates(doc_text: str):
    """逐行解析文档，返回 (lineno, token, cwd) 列表。"""
    candidates: list[tuple[int, str, str | None]] = []
    cwd: str | None = None
    in_fence = False
    for lineno, raw in enumerate(doc_text.splitlines(), start=1):
        stripped = raw.strip()
        if stripped.startswith("```"):
            in_fence = not in_fence
            cwd = None  # 新代码块重置 cd 上下文
            continue
        if not in_fence:
            continue
        m = CD_RE.match(raw)
        if m:
            target = m.group(1).strip("`")
            cwd = None if ("<" in target or ">" in target) else target
        for m in BASH_RE.finditer(raw):
            tok = m.group(1)
            if tok.startswith("-") or "<" in tok or ">" in tok:
                continue
            p = TOKEN_RE.match(tok)
            if p:
                candidates.append((lineno, p.group(0), cwd))
        for m in DOTSLASH_RE.finditer(raw):
            tok = m.group(1)
            if "<" in tok or ">" in tok:
                continue
            p = TOKEN_RE.match(tok)
            if p:
                candidates.append((lineno, "./" + p.group(0), cwd))
    return candidates


def resolve(token: str, cwd: str | None, root: Path) -> Path | None:
    if token.startswith("./"):
        if cwd is None:
            return None
        return root / cwd / token[2:]
    if "/" in token:
        return root / token if cwd is None else root / cwd / token
    if cwd is None:
        return None
    return root / cwd / token


def check_doc(root: Path, doc_rel: str) -> list[str]:
    doc_path = root / doc_rel
    if not doc_path.is_file():
        return [f"{doc_rel}: missing (unable to validate deployment doc paths)"]
    text = doc_path.read_text(encoding="utf-8", errors="replace")
    hits: list[str] = []
    for lineno, token, cwd in parse_candidates(text):
        resolved = resolve(token, cwd, root)
        if resolved is None:
            continue
        if not resolved.exists():
            rel = resolved.relative_to(root)
            hits.append(
                f"{doc_rel}:{lineno}: referenced path {token!r} -> {rel} does not exist"
            )
    return hits


def main(argv: list[str] | None = None) -> int:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--root", type=Path, default=ROOT)
    parser.add_argument("--doc", default=DEFAULT_DOC)
    args = parser.parse_args(argv)
    root = args.root.resolve()
    hits = check_doc(root, args.doc)
    if hits:
        print("VIOLATION (OPT-20260824-018 deployment doc path drift):")
        for h in hits:
            print(f"  - {h}")
        return 1
    print(f"ok: {args.doc} referenced paths exist")
    return 0


if __name__ == "__main__":
    sys.exit(main())
