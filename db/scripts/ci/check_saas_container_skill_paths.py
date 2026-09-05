#!/usr/bin/env python3
"""CI: git-oauth / comment write API paths must appear in saas-machine-container.md §5.2.

OPT-20260822-011 — 之前 PR 回复/一键合并路径已写入 skill 文档，但只有 taskAiProvider
`TestHandleSaasMachineContainerSkill` 的静态字符串断言，挡不住「OpenAPI 加了新接口、
skill 文档忘了同步」的漂移。本检查：

  1. 从 taskGitOauth `src/openapi_merge_request.go` 的 `openAPIMergeRequestPaths()`
     解析全部 `/api/...` 写接口路径（正则提取 map key）。
  2. 从 taskTaskService `src/comment_reply_path.go` 的路径模板注释解析评论回复写接口路径
     （`api/tenant_id/{tid}/workspaceId/{wid}/tasks/{taskId}/comments/{parent}`），
     归一化为文档使用的 `{parent_comment_id}` 模板。
  3. 断言这些路径全部出现在 `docs/skills/saas-container/saas-machine-container.md`。
  4. 任一缺失即退出码 1，新增 git-oauth / 评论写接口时若忘更文档立刻跑红。

用法：
  python3 db/scripts/ci/check_saas_container_skill_paths.py [--root <monorepo>]
"""

from __future__ import annotations

import argparse
import re
import sys
from pathlib import Path

# taskGitOauth 写接口路径来源：openAPIMergeRequestPaths() 的 map key
GITOAUTH_OPENAPI_REL = "taskGitOauth/src/openapi_merge_request.go"
# taskTaskService 评论回复路径模板注释（parseTenantIDCommentReplyPath 顶部）
COMMENT_REPLY_REL = "taskTaskService/src/comment_reply_path.go"
# 厂商门户 skill 文档（taskAiProvider 服务端同源）
SKILL_DOC_REL = "docs/skills/saas-container/saas-machine-container.md"

# 归一化：代码模板 `{parent}` vs 文档模板 `{parent_comment_id}`
COMMENT_PARAM_NORMALIZE = (("{parent}", "{parent_comment_id}"),)


def monorepo_root(start: Path) -> Path:
    here = start.resolve()
    for parent in [here, *here.parents]:
        if (parent / "db" / "registry.yaml").is_file():
            return parent
    raise FileNotFoundError(f"db/registry.yaml not found above {start}")


def extract_openapi_paths(src: str) -> list[str]:
    """提取 `"/api/...": {` 形式的 OpenAPI map key（去除引号）。"""
    paths: list[str] = []
    for m in re.finditer(r'^\s*"(/api/[^"]+)":\s*map\[string\]any', src, re.MULTILINE):
        p = m.group(1)
        if p not in paths:
            paths.append(p)
    return paths


def extract_comment_reply_template(src: str) -> str | None:
    """从注释 `// api/tenant_id/{tid}/.../comments/{parent}` 提取路径模板。"""
    m = re.search(r"//\s*(api/tenant_id/\{tid\}/workspaceId/\{wid\}/tasks/\{taskId\}/comments/\{parent\})", src)
    if not m:
        return None
    raw = m.group(1)
    path = "/" + raw
    if not path.endswith("/"):
        path += "/"
    for old, new in COMMENT_PARAM_NORMALIZE:
        path = path.replace(old, new)
    return path


def normalize_path(p: str) -> str:
    p = p.strip()
    if not p.startswith("/"):
        p = "/" + p
    return p


def main(argv: list[str] | None = None) -> int:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--root", default=None, help="monorepo 根目录（默认向上查找 db/registry.yaml）")
    parser.add_argument("--verbose", action="store_true", help="列出检查到的路径")
    args = parser.parse_args(argv)

    root = monorepo_root(Path(args.root or Path.cwd()))

    gitoauth_file = root / GITOAUTH_OPENAPI_REL
    comment_file = root / COMMENT_REPLY_REL
    doc_file = root / SKILL_DOC_REL

    missing_files = [p for p in (gitoauth_file, comment_file, doc_file) if not p.is_file()]
    if missing_files:
        print(f"ERROR: missing files: {', '.join(str(p.relative_to(root)) for p in missing_files)}", file=sys.stderr)
        return 2

    openapi_paths = extract_openapi_paths(gitoauth_file.read_text(encoding="utf-8"))
    if not openapi_paths:
        print(f"ERROR: no /api/... paths parsed from {GITOAUTH_OPENAPI_REL}", file=sys.stderr)
        return 2

    comment_template = extract_comment_reply_template(comment_file.read_text(encoding="utf-8"))
    if not comment_template:
        print(f"ERROR: comment reply path template not found in {COMMENT_REPLY_REL}", file=sys.stderr)
        return 2

    doc = doc_file.read_text(encoding="utf-8")
    # 归一化文档中的占位符命名差异（{parent_comment_id} vs 代码 {parent}）已在模板侧处理；
    # 文档路径可能带反引号（markdown 行内 code），比较前去掉反引号。
    doc_clean = doc.replace("`", "")

    missing: list[str] = []
    for p in openapi_paths + [comment_template]:
        np = normalize_path(p)
        if np not in doc_clean:
            missing.append(np)

    if args.verbose:
        print(f"git-oauth OpenAPI paths ({len(openapi_paths)}):")
        for p in openapi_paths:
            print(f"  {p}")
        print(f"comment reply path: {comment_template}")

    if missing:
        print("FAIL: skill doc missing write API path(s):", file=sys.stderr)
        for p in missing:
            print(f"  - {p}", file=sys.stderr)
        print(f"请同步更新 {SKILL_DOC_REL} §5.2（新增 git-oauth / 评论写接口必须登记）", file=sys.stderr)
        return 1

    print(f"OK: {len(openapi_paths) + 1} write API paths present in {SKILL_DOC_REL}")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
