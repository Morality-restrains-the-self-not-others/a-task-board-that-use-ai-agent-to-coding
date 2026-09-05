#!/usr/bin/env python3
"""
批量补全 docs/intents 下功能意图文档的「业务意图 → 事件对照」表。

规则 SSOT：.ai/08_prompt_management/01_intent_driven_development.md

用法（仓库根）：
  python3 runAll/scripts/backfill_intent_event_tables.py
  python3 runAll/scripts/backfill_intent_event_tables.py --dry-run
"""

from __future__ import annotations

import argparse
import re
import sys
from pathlib import Path

ROOT = Path(__file__).resolve().parents[2]

INTENT_ROOTS = (
    ROOT / "docs" / "intents",
    ROOT / "task2app" / "docs" / "intents",
)

SECTION_HEADING = "## 业务意图 → 事件对照"

# 路径片段：默认视为无业务事件（可被正文强信号覆盖）
DEFAULT_NO_EVENT_PATH_PARTS = (
    "frontend",
    "design_governance",
)

# 文件名/路径关键词：运维、评估、纯配置，默认无业务事件
INFRA_NO_EVENT_NAME_RE = re.compile(
    r"(bind_0\.0\.0\.0|edge_nginx|high_traffic|Postgres评估|design_md|"
    r"logs_clear_via_truncate|多副本Postgres)",
    re.I,
)

# 正文强信号：即使在 frontend 下也视为需要事件
SERVER_EVENT_SIGNALS = re.compile(
    r"(kafka|send_event|PublishEvent|publishSSE|SSE_MESSAGE|领域事件|"
    r"消息队列|domain.?event|TASK_STATUS_CHANGED|publish\s+SSE|"
    r"投递|消费者)",
    re.I,
)

# 已知意图 → 事件名覆盖（优先于启发式）
KNOWN_EVENT_OVERRIDES: dict[str, list[tuple[str, str, str, str]]] = {
    "022_task_status_changed_release_servers": [
        (
            "任务进度变更为终态",
            "TaskStatusChanged",
            "task 状态更新用例 / Django 或 Go 任务服务",
            "释放关联运行中服务器的 MQ 消费者",
        ),
    ],
    "relay_status_push_cloud_converge": [
        (
            "relay status-push 收敛成功",
            "RelayStatusConverged",
            "taskCloudService handleRelayStatusPush",
            "Kafka SSE_MESSAGE → task-events → task-sse",
        ),
    ],
    "001_budget账本迁Go": [
        (
            "Budget usage 写入成功",
            "BudgetUsageRecorded",
            "taskCloudService /api/internal/budget/*",
            "账本投影 / 控制台只读",
        ),
        (
            "FeatureParams env snapshot 写入",
            "FeatureParamsEnvSnapshotAppended",
            "taskCloudService handleFeatureParamsEnv",
            "容器 env 消费",
        ),
    ],
    "wechat-pay-recharge": [
        (
            "微信充值支付成功回调",
            "WechatPayRechargeSucceeded",
            "支付回调处理 / taskBill 或 accounts",
            "入账、通知消费者",
        ),
    ],
    "fix-billing-account-self-healing": [
        (
            "账单账户自愈完成",
            "BillingAccountSelfHealed",
            "账单自愈用例",
            "审计 / 下游同步",
        ),
    ],
}


def is_functional_intent(path: Path) -> bool:
    name = path.name
    if name.endswith(".test-intent.md") or name.endswith(".test.intent.md"):
        return False
    if not name.endswith(".intent.md"):
        return False
    if "00_索引" in path.parts:
        return False
    if "intent_template" in name:
        return False
    if name.endswith("_DEPLOY.md") or "DEPLOY" in name:
        return False
    return True


def already_has_section(text: str) -> bool:
    return bool(re.search(r"^##\s*业务意图\s*[→\-–—]\s*事件对照\s*$", text, re.M))


def title_from_doc(text: str, path: Path) -> str:
    for line in text.splitlines():
        s = line.strip()
        if s.startswith("# "):
            t = s[2:].strip()
            t = re.sub(r"^NNN_?", "", t)
            t = re.sub(r"（功能意图）$|（测试意图）$|#.*", "", t).strip()
            return t or path.stem
    stem = path.stem
    if stem.endswith(".intent"):
        stem = stem[: -len(".intent")]
    return stem


def stem_key(path: Path) -> str:
    stem = path.stem
    if stem.endswith(".intent"):
        stem = stem[: -len(".intent")]
    return stem


def to_past_tense_event(stem: str, path: Path | None = None) -> str:
    """从文件名启发式生成 PastTense 事件名（存量回填用，可后续人工精修）。"""
    # 去掉序号前缀
    s = re.sub(r"^\d+_", "", stem)
    # 非 ascii 主题：用 目录+序号 保证文档内唯一，意图列保留中文简述
    if re.search(r"[\u4e00-\u9fff]", s):
        num_m = re.match(r"^(\d+)_", stem)
        num = num_m.group(1) if num_m else "0"
        parent = ""
        if path is not None and path.parent.name not in ("intents", "docs"):
            parent = re.sub(r"[^A-Za-z0-9]+", "", path.parent.name.title())
        return f"{parent or 'Doc'}Intent{num}Completed"
    parts = re.split(r"[_\-\s]+", s)
    parts = [p for p in parts if p]
    if not parts:
        return "BusinessIntentCompleted"
    camel = "".join(p[:1].upper() + p[1:] for p in parts)
    # 已是过去式则保留
    if camel.endswith(
        ("ed", "Changed", "Created", "Updated", "Deleted", "Succeeded", "Failed", "Started", "Stopped")
    ):
        return camel
    # 常见动词后缀
    for suf, past in (
        ("Migration", "Migrated"),
        ("Cutover", "CutOver"),
        ("Whitelist", "Whitelisted"),
        ("Release", "Released"),
        ("Converge", "Converged"),
        ("Bootstrap", "Bootstrapped"),
        ("Refresh", "Refreshed"),
        ("Push", "Pushed"),
        ("Start", "Started"),
        ("Stop", "Stopped"),
        ("Create", "Created"),
        ("Update", "Updated"),
        ("Delete", "Deleted"),
        ("Clear", "Cleared"),
        ("Sync", "Synced"),
        ("Bind", "Bound"),
        ("Login", "LoggedIn"),
        ("Recharge", "Recharged"),
    ):
        if camel.endswith(suf):
            return camel[: -len(suf)] + past
    return camel + "Completed"


def infer_publisher(text: str, path: Path) -> str:
    low = text.lower()
    path_s = str(path).replace("\\", "/")
    if "taskcloud" in low or "taskCloudService" in text or "/cloud/" in path_s:
        return "taskCloudService"
    if "taskauth" in low or "taskAuth" in text:
        return "taskAuth"
    if "gateway" in low or "taskContainerGateway" in text:
        return "taskContainerGateway"
    if "go_relay" in low or "relayToTrae" in text:
        return "go_relayToTrae / onlineServiceJS"
    if "kafka" in low or "task-events" in low or "taskEvents" in text:
        return "taskEvents / 事件发布适配器"
    if "django" in low:
        return "Django 应用服务"
    if "/backend/" in path_s:
        return "后端服务（见意图正文）"
    if "/container/" in path_s:
        return "容器/relay 启动路径"
    if "/platform/" in path_s:
        return "runAll / 平台运维"
    if "/engineering/" in path_s:
        return "工程侧服务（见意图正文）"
    return "对应服务应用服务"


def classify_no_event(path: Path, text: str) -> str | None:
    """返回例外理由；None 表示需要事件行。"""
    path_s = str(path).replace("\\", "/")
    if SERVER_EVENT_SIGNALS.search(text):
        return None
    if INFRA_NO_EVENT_NAME_RE.search(path.name) or INFRA_NO_EVENT_NAME_RE.search(path_s):
        return "基础设施/运维/容量评估类意图，不产生业务领域事件"
    parts_lower = {p.lower() for p in path.parts}
    if any(p in parts_lower for p in DEFAULT_NO_EVENT_PATH_PARTS):
        return "纯前端展示/交互或设计治理，无服务端业务状态变更意图"
    # 评估类工程文档
    if "评估" in title_from_doc(text, path) or "评估" in path.name:
        return "评估/调研类意图，无运行时业务事件投递"
    return None


def build_section(path: Path, text: str) -> str:
    title = title_from_doc(text, path)
    key = stem_key(path)
    reason = classify_no_event(path, text)

    lines = [
        "",
        SECTION_HEADING,
        "",
        "> 存量回填（自动）：对照 `.ai/08_prompt_management/01_intent_driven_development.md`。"
        "事件名若为启发式占位，可在后续迭代精修。",
        "",
    ]

    if key in KNOWN_EVENT_OVERRIDES:
        lines.append("| 业务意图 | 事件名（过去式） | 发布点 | 消费者/副作用 | 例外理由 |")
        lines.append("|---------|----------------|--------|--------------|---------|")
        for intent, event, publisher, consumer in KNOWN_EVENT_OVERRIDES[key]:
            lines.append(f"| {intent} | {event} | {publisher} | {consumer} | — |")
        lines.append("")
        return "\n".join(lines)

    if reason:
        lines.append(f"**无对应事件**：{reason}。")
        lines.append("")
        lines.append("| 业务意图 | 事件名（过去式） | 发布点 | 消费者/副作用 | 例外理由 |")
        lines.append("|---------|----------------|--------|--------------|---------|")
        lines.append(f"| {title} | — | — | — | {reason} |")
        lines.append("")
        return "\n".join(lines)

    event = to_past_tense_event(key, path)
    publisher = infer_publisher(text, path)
    # 从正文抽一句意图简述
    intent_label = title
    m = re.search(r"(?:##\s*意图|/s*意图\s*\n)(.+?)(?:\n##|\n\n)", text, re.S)
    if m:
        brief = " ".join(m.group(1).strip().split())[:80]
        if brief:
            intent_label = brief

    consumer = "见意图正文 / 待确认消费者"
    if re.search(r"SSE|sse", text):
        consumer = "SSE / 下游 MQ 消费者"
    elif re.search(r"邮件|email", text, re.I):
        consumer = "邮件发送 Handler"
    elif re.search(r"释放|release", text, re.I):
        consumer = "资源释放消费者"

    lines.append("| 业务意图 | 事件名（过去式） | 发布点 | 消费者/副作用 | 例外理由 |")
    lines.append("|---------|----------------|--------|--------------|---------|")
    lines.append(f"| {intent_label} | {event} | {publisher} | {consumer} | — |")
    lines.append("")
    return "\n".join(lines)


def insert_section(text: str, section: str) -> str:
    # 插在「变更记录」之前；否则文末
    for pat in (
        r"(^##\s*变更记录\s*$)",
        r"(^##\s*5\.\s*变更记录\s*$)",
        r"(^##\s*变更要点\s*$)",
    ):
        m = re.search(pat, text, re.M)
        if m:
            return text[: m.start()] + section + text[m.start() :]
    if not text.endswith("\n"):
        text += "\n"
    return text + section


def iter_intent_files() -> list[Path]:
    out: list[Path] = []
    for root in INTENT_ROOTS:
        if not root.is_dir():
            continue
        for p in sorted(root.rglob("*.intent.md")):
            if is_functional_intent(p):
                out.append(p)
    return out


def update_template() -> bool:
    tpl = ROOT / "task2app" / "docs" / "intents" / "00_索引" / "intent_template.intent.md"
    if not tpl.is_file():
        return False
    text = tpl.read_text(encoding="utf-8")
    if already_has_section(text):
        return False
    block = f"""
{SECTION_HEADING}

| 业务意图 | 事件名（过去式） | 发布点 | 消费者/副作用 | 例外理由 |
|---------|----------------|--------|--------------|---------|
| （本意图的服务端业务动作） | PastTenseEventName | 应用服务 / 发布点 | 消费者或副作用 | — 或填写无事件理由 |

> 纯查询/纯前端：写 **无对应事件** 及理由，事件名列填 `—`。
> 细则：`.ai/08_prompt_management/01_intent_driven_development.md`
"""
    # 插在验收标准之后或实施计划之前
    m = re.search(r"(^##\s*5\.\s*实施计划\s*$)", text, re.M)
    if m:
        new = text[: m.start()] + block + "\n" + text[m.start() :]
    else:
        new = text.rstrip() + "\n" + block + "\n"
    tpl.write_text(new, encoding="utf-8")
    return True


def main() -> int:
    parser = argparse.ArgumentParser()
    parser.add_argument("--dry-run", action="store_true")
    args = parser.parse_args()

    files = iter_intent_files()
    updated = 0
    skipped = 0
    for path in files:
        text = path.read_text(encoding="utf-8")
        if already_has_section(text):
            skipped += 1
            continue
        section = build_section(path, text)
        new_text = insert_section(text, section)
        rel = path.relative_to(ROOT)
        if args.dry_run:
            print(f"WOULD UPDATE {rel}")
        else:
            path.write_text(new_text, encoding="utf-8")
            print(f"UPDATED {rel}")
        updated += 1

    tpl_changed = False
    if not args.dry_run:
        tpl_changed = update_template()

    print(
        f"\nDone: updated={updated} skipped_existing={skipped} "
        f"template_updated={tpl_changed} total_scanned={len(files)}"
    )
    return 0


if __name__ == "__main__":
    sys.exit(main())
