#!/usr/bin/env python3
"""OPT-20260820-005 回归门禁：注销阻断项 action_url 必须命中 taskFE 已注册 Vue 路由。

2026-08-20：注销阻断「前往处理」曾指向未注册路径（`/billing/gitlab-resources/`、
`/settings/members/`、`/workspace/.../task/`），SPA catch-all 会把用户踢回首页。
后端已改为下发已注册路径，前端 remap（`accountDeletionActionUrl.js`）+ 路由别名
兜底旧路径。

本门禁静态校验三个服务（taskBill / taskCloudService / taskTenantService）的
`account_deletion_action_url.go` 产出的所有 URL 模式：

1. 从 `taskFE/app/src/router/*.js` 收集 `path: '...'` 已注册路由，以及带
   `redirect:` 的别名路由（别名最终重定向到已注册页面，算合法）。
2. 从 `account_deletion_action_url.go` 提取 `fmt.Sprintf("/...")` /
   `return "/..."` 字面量（`%s` → 任意路径段，`%d` → 数字段）。
3. 每个后端 URL 模式必须能实例化出至少一个命中「已注册路由或别名」的 URL，
   否则阻断（防止新增 blocker 再写假路径）。

用法：`python3 db/scripts/ci/check_account_deletion_action_urls.py`
exit 0 = 通过；exit 1 = 存在后端可能下发但前端未注册（或未别名）的路径。
"""
from __future__ import annotations

import re
import sys
from pathlib import Path

ROOT = Path(__file__).resolve().parents[3]
ROUTER_DIR = ROOT / "taskFE" / "app" / "src" / "router"
ACTION_URL_FILES = [
    ROOT / "taskBill" / "src" / "account_deletion_action_url.go",
    ROOT / "taskCloudService" / "src" / "account_deletion_action_url.go",
    ROOT / "taskTenantService" / "src" / "account_deletion_action_url.go",
]

PATH_RE = re.compile(r"path:\s*'([^']+)'")
SPRINTF_RE = re.compile(r'fmt\.Sprintf\(\s*"([^"]+)"\s*,')
RETURN_STR_RE = re.compile(r'return\s+"(/[^"]*)"')


def route_to_regex(path: str) -> str:
    """Vue 路由 path → 正则（`:param` 段 → `[^/]+`）。"""
    out: list[str] = []
    i = 0
    n = len(path)
    while i < n:
        if path[i] == ":":
            j = i + 1
            while j < n and path[j] != "/":
                j += 1
            out.append(r"[^/]+")
            i = j
        else:
            j = i
            while j < n and path[j] != ":":
                j += 1
            out.append(re.escape(path[i:j]))
            i = j
    return "^" + "".join(out) + "$"


def go_url_sample(pattern: str) -> str:
    """Go fmt 模式 → 代表性具体 URL（`%s`→'x'，`%d`→'1'）。"""
    out: list[str] = []
    i = 0
    n = len(pattern)
    while i < n:
        if pattern[i] == "%" and i + 1 < n:
            out.append("1" if pattern[i + 1] == "d" else "x")
            i += 2
        else:
            out.append(pattern[i])
            i += 1
    return "".join(out)


def collect_router_paths() -> tuple[list[str], list[str]]:
    """返回 (已注册路由, 别名路由)。别名 = 同一路由对象带 `redirect:` 的 path。"""
    registered: list[str] = []
    aliases: list[str] = []
    for f in sorted(ROUTER_DIR.glob("*.js")):
        src = f.read_text(encoding="utf-8")
        matches = list(PATH_RE.finditer(src))
        for idx, m in enumerate(matches):
            end = matches[idx + 1].start() if idx + 1 < len(matches) else len(src)
            segment = src[m.end() : end]
            if re.search(r"\bredirect\s*:", segment):
                aliases.append(m.group(1))
            else:
                registered.append(m.group(1))
    return registered, aliases


def collect_action_url_patterns() -> list[tuple[Path, str]]:
    """返回 [(源文件, Go fmt 模式)]：注销阻断可下发的 action_url 模式。"""
    found: list[tuple[Path, str]] = []
    for f in ACTION_URL_FILES:
        if not f.exists():
            continue
        src = f.read_text(encoding="utf-8")
        for m in SPRINTF_RE.finditer(src):
            found.append((f, m.group(1)))
        for m in RETURN_STR_RE.finditer(src):
            found.append((f, m.group(1)))
    return found


def _all_route_regexes() -> list[re.Pattern[str]]:
    registered, aliases = collect_router_paths()
    return [re.compile(route_to_regex(p)) for p in registered + aliases]


def check_all() -> list[tuple[Path, str]]:
    """返回未命中任何已注册路由/别名的 [(源文件, 模式)]。空列表 = 通过。"""
    route_res = _all_route_regexes()
    failures: list[tuple[Path, str]] = []
    for src_file, pattern in collect_action_url_patterns():
        sample = go_url_sample(pattern)
        if not any(rx.fullmatch(sample) for rx in route_res):
            failures.append((src_file, pattern))
    return failures


def main() -> int:
    failures = check_all()
    if not failures:
        patterns = collect_action_url_patterns()
        print(f"OK: {len(patterns)} 个注销阻断 action_url 模式均命中已注册 Vue 路由。")
        return 0
    for src_file, pattern in failures:
        rel = src_file.relative_to(ROOT)
        print(
            f"FAIL {rel}: 后端可能下发 {pattern!r}，taskFE 无对应已注册路由或别名（SPA catch-all 会踢回首页）。",
            file=sys.stderr,
        )
    print(f"{len(failures)} 个 action_url 模式未命中任何已注册 Vue 路由。", file=sys.stderr)
    return 1


if __name__ == "__main__":
    sys.exit(main())
