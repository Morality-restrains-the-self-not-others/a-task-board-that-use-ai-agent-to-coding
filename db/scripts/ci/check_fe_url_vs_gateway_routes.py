#!/usr/bin/env python3
"""OPT-20260807-037 回归门禁：taskFE /api/ URL 模板须被 taskGateway routes.yaml 路由覆盖。

背景：1508f24 将 billing 前端路径改为 `/api/tenant/{tid}/billing/*`（位置式
parseTenantID 契约），但 taskGateway routes.yaml 未同步新增该前缀路由 → 请求落入
task-tenant-service(863) 兜底 `/api/tenant/*` → 404 `{"error":"not found"}`，
billing 模块整体静默失效两天。前端路径迁移类改动须核对网关路由前缀覆盖。

本门禁静态扫描 taskFE app/src 的 .js/.vue/.ts 中所有含 `/api/` 的字符串字面量：

1. 解析 taskGateway/routes/routes.yaml，提取全部路由 uri（支持 `uri` 单值与 `uris`
   列表），排除 null-upstream（api-orphaned-not-found → 立即 404）兜底路由。
2. 对每个 FE URL 模板，去掉查询串/锚点（感知 `${...}` 插值），把 `{param}` 与 `${expr}`
   归一化为 glob 通配 `*`；若没有任何非兜底网关路由能 glob 覆盖（fnmatch）→ 阻断。

豁免（与 check_fe_url_placeholder.py / check_fe_url_trailing_slash.py 一致）：
- 注释内字符串、`it()`/`test()`/`describe()` 描述串、`.replace()`/`.replaceAll()` 替换串
- `*.test.*` / `*.spec.*` 文件（测试 mock 非真实调用点）
- ALLOWLIST：已知有意不经过网关的 URL（孤儿页面保留路径 / 表单配置默认值）

用法：pre-commit 门禁 `python3 db/scripts/ci/check_fe_url_vs_gateway_routes.py`
exit 0 = 通过；exit 1 = 命中未被网关覆盖的 URL（列出文件 + 模板）。
"""
import os
import re
import sys
from pathlib import Path
from fnmatch import fnmatch

try:
    import yaml
except ImportError:
    yaml = None

ROOT = Path(__file__).resolve().parents[3]
FE_SRC = str(ROOT / "taskFE" / "app" / "src")
ROUTES_FILE = ROOT / "taskGateway" / "routes" / "routes.yaml"

EXTENSIONS = (".js", ".vue", ".ts")

# 已知有意不经过网关的 URL（归一化 glob 前缀匹配）：
# - SystemAdminCloudAuthorizations 孤儿页面：未挂载 router，taskCloudService 无
#   server-images 处理器（OPT-20260806-002），路径保留以便功能恢复时定位。
# - SystemAdminSubTokenProviders 表单「派生端点路径」默认值：配置数据，非请求目标。
ALLOWLIST_PREFIXES = (
    "/api/system-admin/*/cloud/server-images",
    "/api/token/derive",
)

# 字符串字面量（单/双引号 + 反引号模板，非贪婪）
STRING_RE = re.compile(r"""(['"`])(.*?)(?<!\\)\1""", re.DOTALL)

TEST_FN_RE = re.compile(r"\b(?:it|test|describe|skip|only)\($")
REPLACE_RE = re.compile(r"\.replace(?:All)?\(")

_INTERP_RE = re.compile(r"\$\{[^}]*\}")
_PARAM_RE = re.compile(r"\{[^}]*\}")


def _comment_spans(src: str):
    """计算注释区间 [(start, end)]，跳过字符串内的 // 与 /*。"""
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
    return bool(REPLACE_RE.search(before))


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


def fe_normalize(url: str) -> str:
    """归一化 FE URL 模板：去查询/锚点，`{param}`/`${expr}` → `*`。"""
    path = strip_query(url)
    path = _INTERP_RE.sub("*", path)
    path = _PARAM_RE.sub("*", path)
    return path


def load_gateway_uris(routes_file=ROUTES_FILE):
    """解析 routes.yaml → (cover_uris, fallback_uris)。

    cover_uris: 指向真实 upstream 的路由 uri（glob 模板）。
    fallback_uris: null-upstream（api-orphaned-not-found）路由 uri。
    """
    if not routes_file.is_file():
        return [], []
    if yaml is None:
        print("⚠ 缺少 PyYAML，跳过网关路由解析", file=sys.stderr)
        return [], []
    data = yaml.safe_load(routes_file.read_text(encoding="utf-8"))
    routes = data.get("routes", []) or []
    cover_uris = []
    fallback_uris = []
    for r in routes:
        ups = r.get("upstream")
        uris = r.get("uris") or r.get("uri") or []
        if isinstance(uris, str):
            uris = [uris]
        for u in uris:
            u = str(u)
            if not u.startswith("/api/"):
                continue
            if ups == "null-upstream":
                fallback_uris.append(u)
            else:
                cover_uris.append(u)
    return cover_uris, fallback_uris


def is_covered(fe_url: str, cover_uris) -> bool:
    """FE URL 模板是否被任一网关路由 glob 覆盖。"""
    fe = fe_normalize(fe_url)
    if not fe.startswith("/api/"):
        return True  # 非 /api/ 不归网关管
    for prefix in ALLOWLIST_PREFIXES:
        if fe.startswith(prefix):
            return True
    for gu in cover_uris:
        if fnmatch(fe, gu):
            return True
    return False


def scan_file(path: str, cover_uris, hits: list, rel: str):
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
        if not is_covered(content, cover_uris):
            hits.append((rel, content.strip()[:120]))


def main():
    target = sys.argv[1] if len(sys.argv) > 1 else FE_SRC
    if not os.path.isdir(target):
        print(f"skip: {target} 不存在", file=sys.stderr)
        return 0
    cover_uris, fallback_uris = load_gateway_uris()
    if not cover_uris:
        print("⚠ 未解析到网关路由 uri，跳过", file=sys.stderr)
        return 0
    hits = []
    for dirpath, _dirs, files in os.walk(target):
        for name in sorted(files):
            if not name.endswith(EXTENSIONS):
                continue
            if ".test." in name or ".spec." in name:
                continue
            p = os.path.join(dirpath, name)
            rel = os.path.relpath(p, ROOT)
            scan_file(p, cover_uris, hits, rel)
    if hits:
        print(f"⛔ OPT-20260807-037 taskFE URL 未被网关路由覆盖 {len(hits)} 处"
              f"（网关 {len(cover_uris)} 条有效路由，{len(fallback_uris)} 条兜底除外）：")
        for rel, content in hits:
            print(f"    {rel}: {content}")
        print("    前端路径迁移类改动须同步 taskGateway/routes/routes.yaml 路由前缀；"
              "确属孤儿/配置数据可在脚本 ALLOWLIST_PREFIXES 登记。")
        return 1
    print(f"✓ OPT-20260807-037 taskFE URL 全覆盖网关路由（{len(cover_uris)} 条有效路由，0 处缺口）")
    return 0


if __name__ == "__main__":
    sys.exit(main())
