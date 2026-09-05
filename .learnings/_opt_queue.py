"""OPT queue parse / classify / blocked-file helpers.

Used by session agents to keep OPTIMIZATION_TODOS.md locally executable,
and to park blocked items in BLOCK_TODO_<CATEGORY>.md.
"""
from __future__ import annotations

import re
from dataclasses import dataclass

BLOCK_CATEGORIES = ("BROWSER", "OPS", "INFRA")
ALL_DESTINATIONS = ("OPEN", "BROWSER", "OPS", "INFRA", "PRODUCT")

_ITEM_RE = re.compile(
    r"^### (?P<id>OPT-\d{8}-\d+)\s*[—:][ \t]*(?P<title>[^\n]*)\n(?P<body>.*?)(?=^### OPT-|\Z)",
    re.MULTILINE | re.DOTALL,
)

_PRODUCT_MARKERS = (
    "待产品决策",
    "PRODUCT_DECISIONS.md",
    "已转出，待产品决策",
)

_INFRA_MARKERS = (
    "minio",
    "无对象存储",
    "腾讯云 cos",
    "cos-go-sdk",
    "tencent/cos",
    "正式 logo",
    "正式品牌",
    "品牌方标",
)

_OPS_MARKERS = (
    "厂商门户重生",
    "厂商/管理门户",
    "在厂商门户",
    "对该 ubuntu 模板",
    "重生 userdata",
    "重生并发布",
    "重生模板",
    "待模板重生",
    "清库待人工",
    "清空+初始化",
    "清空全部数据库",
    "initalldatabases",
    "云安全组",
    "需云安全组",
    "重建 gitlab 容器",
    "github app",
    "安装到 task2money",
    "重存 linux userdata",
)

_BROWSER_TITLE_MARKERS = (
    "公网验收",
    "公网硬刷新",
    "公网精准重启",
    "公网登录态",
    "公网任务详情",
)

_BROWSER_STATUS_MARKERS = (
    "待浏览器",
    "待登录 cookie",
    "待真机",
    "待多公司账号",
    "待 cdp",
    "部署完成，待浏览器",
)

_BROWSER_MARKERS = (
    "硬刷新",
    "待浏览器",
    "待登录 cookie",
    "cdp 9222",
    "cdp +",
    "待真机",
    "待多公司账号",
    "浏览器登录态",
    "登录后手验",
    "浏览器复验",
    "精准重启后验收",
    "精准编译重启后",
    "待 cdp",
    "playwright/生产验收",
    "生产实测",
    "目视验收",
)

_OPEN_OVERRIDE_MARKERS = (
    "playwright：",
    "新增 playwright",
    "不得无 start-all",
    "排查 createproject e2e",
    "排查 shutdown-self",
    "容器 8080 就绪",
    "合并 people.access",
    "邀请页改为选择可复用角色",
    "默认服务器配置表单补充",
    "统一「智能体资源设置」",
)


@dataclass(frozen=True)
class OptItem:
    opt_id: str
    title: str
    body: str

    def heading_line(self) -> str:
        return f"### {self.opt_id} — {self.title}"

    def full_text(self) -> str:
        body = self.body.rstrip() + "\n"
        return f"{self.heading_line()}\n{body}"


def parse_opt_items(text: str) -> list[OptItem]:
    items: list[OptItem] = []
    for m in _ITEM_RE.finditer(text):
        items.append(
            OptItem(
                opt_id=m.group("id"),
                title=m.group("title").strip(),
                body=m.group("body"),
            )
        )
    return items


def _haystack(title: str, body: str) -> str:
    return f"{title}\n{body}".lower()


def classify_opt(title: str, body: str) -> str:
    """Return OPEN | BROWSER | OPS | INFRA | PRODUCT."""
    hay = _haystack(title, body)
    status_line = ""
    for line in body.splitlines():
        if "**status**" in line.lower():
            status_line = line.lower()
            break

    if any(m.lower() in status_line or m.lower() in hay for m in _PRODUCT_MARKERS):
        # Full executable items that merely mention the product file as history
        # still count as PRODUCT when status says 待产品决策 / 已转出.
        if "待产品决策" in status_line or "已转出" in status_line:
            return "PRODUCT"
        if "product_decisions.md" in hay and "已转出" in hay:
            return "PRODUCT"

    if any(m in hay for m in _OPEN_OVERRIDE_MARKERS):
        return "OPEN"

    title_l = title.lower()
    if any(m in title_l for m in _BROWSER_TITLE_MARKERS):
        return "BROWSER"
    if any(m in status_line for m in _BROWSER_STATUS_MARKERS):
        return "BROWSER"

    if any(m in hay for m in _INFRA_MARKERS):
        return "INFRA"

    if any(m in hay for m in _OPS_MARKERS):
        return "OPS"

    if any(m in hay for m in _BROWSER_MARKERS):
        return "BROWSER"

    return "OPEN"


def split_items_by_category(items: list[OptItem]) -> dict[str, list[OptItem]]:
    grouped = {k: [] for k in ALL_DESTINATIONS}
    for item in items:
        cat = classify_opt(item.title, item.body)
        if cat not in grouped:
            raise ValueError(f"unknown category {cat} for {item.opt_id}")
        grouped[cat].append(item)
    return grouped


def block_todo_filename(category: str) -> str:
    if category not in BLOCK_CATEGORIES:
        raise ValueError(f"not a block category: {category}")
    return f"BLOCK_TODO_{category}.md"


_CATEGORY_LABELS = {
    "BROWSER": "浏览器验收（公网硬刷新 / 登录 Cookie / CDP；优先 Playwright 闭环）",
    "OPS": "运维与破坏性操作（清库、厂商门户重生模板、云安全组、重建容器、外部安装）",
    "INFRA": "基础设施或外部资产（腾讯云 COS、品牌物料等新组件）",
}


def _ensure_blocked_by(body: str, category: str) -> str:
    if "**Blocked-By**:" in body:
        return body
    # Insert after Status line when present.
    lines = body.splitlines(keepends=True)
    out: list[str] = []
    inserted = False
    for line in lines:
        out.append(line)
        if (not inserted) and re.match(r"^-\s*\*\*Status\*\*:", line):
            out.append(f"- **Blocked-By**: {category}\n")
            inserted = True
    if not inserted:
        out.insert(0, f"- **Blocked-By**: {category}\n")
    return "".join(out)


def render_block_todo_file(category: str, items: list[OptItem]) -> str:
    if category not in BLOCK_CATEGORIES:
        raise ValueError(f"not a block category: {category}")
    label = _CATEGORY_LABELS[category]
    parts = [
        f"# Blocked TODOs — {category}\n",
        "\n",
        f"> 本文件存放因 **{label}** 阻塞而从开放清单分流的 OPT 条目。\n",
        "> 阻塞解除后移回 [OPTIMIZATION_TODOS.md](./OPTIMIZATION_TODOS.md) 执行，或完成后迁入 [OPTIMIZATION_TODOS_COMPLETED.md](./OPTIMIZATION_TODOS_COMPLETED.md)。\n",
        "> 分流规则见 [OPTIMIZATION_TODOS.ai.md](./OPTIMIZATION_TODOS.ai.md)「阻塞项分流」。\n",
    ]
    if category == "BROWSER":
        parts.append(
            "> **解除方式**：优先用 Playwright（CDP 9222 / 登录 helper）验收，见 [OPTIMIZATION_TODOS.ai.md](./OPTIMIZATION_TODOS.ai.md)「BROWSER 阻塞项的 Playwright 解除」。\n",
        )
    parts.extend(
        [
            "\n",
            f"- **Category**: `{category}`\n",
            f"- **Count**: {len(items)}\n",
            "\n",
            "---\n",
            "\n",
        ]
    )
    for item in items:
        body = _ensure_blocked_by(item.body, category)
        parts.append(f"### {item.opt_id} — {item.title}\n")
        parts.append(body if body.endswith("\n") else body + "\n")
        if not parts[-1].endswith("\n\n"):
            parts.append("\n")
    return "".join(parts)


def open_list_violations(text: str) -> list[str]:
    """Return violations if an open-list file contains blocked items."""
    bad: list[str] = []
    for item in parse_opt_items(text):
        cat = classify_opt(item.title, item.body)
        if cat != "OPEN":
            bad.append(f"{item.opt_id} classified as {cat}, must not stay in OPTIMIZATION_TODOS.md")
        if "**Blocked-By**:" in item.body:
            bad.append(f"{item.opt_id} has Blocked-By in open list")
    return bad
