#!/usr/bin/env python3
"""
精修存量意图文档中的启发式事件名，并统一「无对应事件」例外。

- 迁表 / 删代码 / 纯配置 / 纯日志：改为书面无事件例外
- 真实业务意图：对齐仓库已有 MQ 类型或 domain/events 契约名

用法（仓库根）：
  python3 runAll/scripts/refine_intent_event_tables.py
  python3 runAll/scripts/refine_intent_event_tables.py --dry-run
"""

from __future__ import annotations

import argparse
import re
import sys
from pathlib import Path

ROOT = Path(__file__).resolve().parents[2]
SECTION_RE = re.compile(r"^##\s*业务意图\s*[→\-–—]\s*事件对照\s*$", re.M)

# stem → 无事件理由（覆盖整段对照表）
NO_EVENT: dict[str, str] = {
    "all_services_bind_0.0.0.0": "部署监听配置变更，不产生业务领域事件",
    "edge_nginx_loopback_bind": "边缘 Nginx 绑定配置，不产生业务领域事件",
    "high_traffic_hardening": "容量护栏/配置加固，不产生业务领域事件",
    "runall_logs_clear_via_truncate": "运维日志 truncate API，不产生业务领域事件",
    "budget_record_usage_batch_thin_removed": "删除 Django thin 壳，无新增业务事件投递",
    "rewrite_sub_token_python_removed": "删除 Python 实现，无新增业务事件投递",
    "saas_budget_ledger_http_cutover": "切断直连的 HTTP cutover，无新增业务事件",
    "task_cloud_saas_sqlite_http_cutover": "切断 SQLite 直连的 HTTP cutover，无新增业务事件",
    "task_events_saas_http_cutover": "切断 saas 直连的 cutover，无新增业务事件",
    "cloud_domain_tables_task_cloud_migration": "表归属迁移决策/验收，无新增业务事件",
    "ai_endpoint_budget_direct_cloud": "API 路由迁 Go（HTTP），无新增业务事件",
    "ai_endpoint_django_thin_to_go": "Django thin 迁 Go（HTTP），无新增业务事件",
    "budget_console_cloud_admin_api": "控制台 API 迁 Cloud（HTTP），无新增业务事件",
    "budget_console_user_api_cloud": "控制台用户 API 迁 Cloud（HTTP），无新增业务事件",
    "container_exec_bypass_django": "热路径路由绕过 Django（HTTP/Gateway），无新增业务事件",
    "container_task_api_via_gateway": "换票路径改经 Gateway（HTTP），无新增业务事件",
    "feature_params_proxy_rewrite_go": "proxy rewrite 逻辑迁 Go（HTTP），无新增业务事件",
    "tenant_budget_permission_cloud": "权限数据迁 Cloud（HTTP/存储），无新增业务事件",
    "token_init_path_scope": "token-init URL/绑定范围配置，无新增业务事件",
    "009_scoped_container_ui_path": "容器 UI path 约定，无新增业务事件",
    "008_selected_image_pull_hash_log": "启动日志增强（可观测性），无业务领域事件投递",
    "006_容器feature-params-env拉取日志": "拉取日志增强（可观测性），无业务领域事件投递",
    "002_budget账本独立库": "账本物理库拆分（基础设施），无新增业务事件",
    "003_budget多副本Postgres评估": "评估/调研文档，无运行时业务事件",
    "onlineServiceJS_bootstrap_transient_retry": "瞬时断连重试修复，无新增业务事件契约",
    "sso_ai_provider_admin_logged_in_no_login_redirect": "SSO 网关认证路径修复，无新增业务事件",
    "gitlab_taskauth_sso_cross_subdomain_login": "跨子域 Cookie/登录修复，无新增业务事件",
    "ai_provider_oidc_issuer_gateway": "OIDC issuer URL 配置，无新增业务事件",
    "001_引入_design_md统一页面设计": "设计治理，无运行时业务事件",
    "fix-billing-account-self-healing": "懒初始化安全网（同步写库），无 MQ 业务事件投递",
    "010_relay_clear_state_on_container_start": "selected_image 启动前清空本机状态为进程内副作用，不产生业务领域事件",
}

# stem → rows: (intent, past_tense_event, mq_or_contract, publisher, consumer)
# mq_or_contract：写入「MQ类型/契约」列，供第二层证据检索
EVENT_ROWS: dict[str, list[tuple[str, str, str, str, str]]] = {
    "022_task_status_changed_release_servers": [
        (
            "任务进度变更为终态",
            "TaskStatusChanged",
            "TASK_STATUS_CHANGED",
            "taskTaskService / 任务状态更新用例",
            "task-events taskstatuschanged → 终态释放服务器",
        ),
    ],
    "023_terminal_hard_release_container_migrate": [
        (
            "终态硬释放机器节点并迁移他任务容器",
            "ContainerMigrateAwaitReady",
            "CONTAINER_MIGRATE_AWAIT_READY",
            "任务终态释放编排",
            "task-events containermigrateawaitready",
        ),
        (
            "关联云服务器停止",
            "CloudServerStopped",
            "CLOUD_SERVER_STOPPED",
            "释放路径 / taskCloudService",
            "云资源回收消费者",
        ),
    ],
    "relay_status_push_cloud_converge": [
        (
            "relay status-push 收敛成功",
            "RelayStatusConverged",
            "relay_status_converged / SSE_MESSAGE",
            "taskCloudService handleRelayStatusPush",
            "Kafka SSE_MESSAGE → task-events → task-sse",
        ),
    ],
    "relay_startup_session_gateway_direct_write": [
        (
            "token-init 成功写入 startup session",
            "RelayTokenInitSucceeded",
            "RELAY_START_ACCEPTED / relay_token_init_succeeded",
            "taskContainerGateway → Cloud upsert",
            "relay 生命周期消费者",
        ),
        (
            "start accept 成功写入 startup session",
            "RelayStartAccepted",
            "RELAY_START_ACCEPTED",
            "taskContainerGateway → Cloud upsert",
            "relay 生命周期消费者",
        ),
    ],
    "wechat-pay-recharge": [
        (
            "微信充值支付成功入账",
            "BillingTransactionCreated",
            "BILLING_TRANSACTION_CREATED",
            "billing_bridge / 支付回调",
            "task-events billing_transaction_created",
        ),
    ],
    "001_budget账本迁Go": [
        (
            "Budget usage 写入成功",
            "BudgetUsageRecorded",
            "BUDGET_USAGE_RECORDED（planned）",
            "taskCloudService /api/internal/budget/*",
            "账本投影（同步写）；证据豁免：HTTP 直写非 MQ；登记 budget-usage-recorded-http",
        ),
        (
            "FeatureParams env snapshot 写入",
            "FeatureParamsEnvSnapshotAppended",
            "FEATURE_PARAMS_ENV_SNAPSHOT_APPENDED（planned）",
            "taskCloudService handleFeatureParamsEnv",
            "容器 env（同步写）；证据豁免：HTTP 直写非 MQ；登记 feature-params-env-snapshot-http",
        ),
    ],
    "create_task_default_top_order": [
        (
            "创建任务并置于看板列首",
            "DomainOrderReordered",
            "domain_order_reordered_event",
            "任务创建用例 / valueStream 或任务服务",
            "看板排序投影",
        ),
    ],
    "container_layer_git_merge": [
        (
            "容器层合并到目标分支成功",
            "ProjectUpdated",
            "PROJECT_UPDATED",
            "容器 Git merge / Gateway",
            "project_updated 消费者",
        ),
    ],
    "multirepo_prefer_remote_gitlab_push_oauth": [
        (
            "多仓 GitLab 推送补齐 OAuth 凭证",
            "RepoCloneCredentialsFetchSucceeded",
            "repo_clone_credentials_fetch_succeeded",
            "容器推送 / OAuth token 同步",
            "凭证缓存与后续 clone/push",
        ),
    ],
    "006_ssh_https_clone_and_selected_image_token_sync": [
        (
            "SSH 仓转 HTTPS 克隆并同步 selected_image token",
            "RepoCloneCredentialsFetchSucceeded",
            "repo_clone_credentials_fetch_succeeded",
            "go_relayToTrae / onlineServiceJS",
            "克隆与 token 缓存",
        ),
        (
            "OAuth token 拉取失败可观测",
            "OauthTokenFetchFailed",
            "oauth_token_fetch_failed",
            "cloud 凭证路径",
            "审计 / 重试",
        ),
    ],
    "007_reclone_https_and_ui_stale_token": [
        (
            "reclone HTTPS 规范化成功",
            "RepoCloneCredentialsFetchSucceeded",
            "repo_clone_credentials_fetch_succeeded",
            "容器/relay 启动路径",
            "克隆消费者",
        ),
        (
            "UI 陈旧 token 刷新",
            "ContainerUiContextRefreshed",
            "container_ui_context_refreshed",
            "onlineServiceJS / Cloud",
            "前端上下文刷新",
        ),
    ],
    "004_start_vm_traceid_equals_task_id": [
        (
            "启动 VM 过程推送进度 SSE",
            "SseMessagePublished",
            "SSE_MESSAGE",
            "taskCloudService compute_start_vm_* / start_vm",
            "task-sse",
        ),
        (
            "云服务器启动成功",
            "CloudServerStarted",
            "CLOUD_SERVER_STARTED",
            "start_vm 成功路径",
            "cloud_server_started 消费者",
        ),
    ],
    "005_create_task_auto_run_backend_start": [
        (
            "创建任务后自动启动运行时",
            "CloudServerStartAuto",
            "CLOUD_SERVER_START_AUTO",
            "创建任务 auto_run 后端",
            "cloud_server_start_auto 消费者",
        ),
    ],
    "006_auto_run_first_instruction_and_delivery": [
        (
            "自动运行首条指令与交付",
            "TaskCommentImageMentioned",
            "TASK_COMMENT_IMAGE_MENTIONED",
            "auto_run 编排",
            "提及驱动启动 / 交付消费者",
        ),
    ],
    "006_auto_sg_ingress_whitelist": [
        (
            "自动创建安全组入网白名单",
            "CloudPlatformAuthorizationCreated",
            "CLOUD_PLATFORM_AUTHORIZATION_CREATED",
            "taskCloudService 安全组编排",
            "云授权/安全组消费者",
        ),
    ],
    "userdata_boot_progress_sse": [
        (
            "UserData 启动步骤进度上报",
            "SseMessagePublished",
            "SSE_MESSAGE",
            "taskCloudService boot-progress → server-startup-status-sse",
            "task-sse / 任务详情进度条",
        ),
    ],
    "001_工作空间机器节点闲置策略": [
        (
            "工作空间机器节点进入闲置",
            "WorkspaceMachineIdle",
            "WORKSPACE_MACHINE_IDLE",
            "闲置策略检测",
            "task-events 回收闲置节点",
        ),
    ],
}

EXEMPT_MARKER = "证据豁免"


def stem_of(path: Path) -> str:
    s = path.stem
    if s.endswith(".intent"):
        s = s[: -len(".intent")]
    return s


def title_of(text: str, path: Path) -> str:
    for line in text.splitlines():
        if line.startswith("# "):
            return line[2:].strip()
    return stem_of(path)


def build_no_event_section(title: str, reason: str) -> str:
    return f"""
## 业务意图 → 事件对照

> 精修（2026-07-15）：对照 `.ai/08_prompt_management/01_intent_driven_development.md`。

**无对应事件**：{reason}。

| 业务意图 | 事件名（过去式） | MQ类型/契约 | 发布点 | 消费者/副作用 | 例外理由 |
|---------|----------------|------------|--------|--------------|---------|
| {title} | — | — | — | — | {reason} |
"""


def build_event_section(rows: list[tuple[str, str, str, str, str]]) -> str:
    lines = [
        "",
        "## 业务意图 → 事件对照",
        "",
        "> 精修（2026-07-15）：事件名对齐仓库 MQ / domain events；"
        f"同步写路径或非 MQ 副作用在例外理由标注「{EXEMPT_MARKER}」。",
        "",
        "| 业务意图 | 事件名（过去式） | MQ类型/契约 | 发布点 | 消费者/副作用 | 例外理由 |",
        "|---------|----------------|------------|--------|--------------|---------|",
    ]
    for intent, event, mq, pub, cons in rows:
        exempt = EXEMPT_MARKER if (EXEMPT_MARKER in cons or mq.strip() in ("—", "-", "")) else "—"
        if EXEMPT_MARKER in cons:
            exempt = cons if EXEMPT_MARKER in cons else exempt
        # If consumer already has 证据豁免, put short form in 例外理由
        if EXEMPT_MARKER in cons:
            reason = cons if "证据豁免" in cons else f"{EXEMPT_MARKER}：非 MQ"
            lines.append(f"| {intent} | {event} | {mq} | {pub} | {cons.split('；')[0]} | {reason} |")
        else:
            lines.append(f"| {intent} | {event} | {mq} | {pub} | {cons} | — |")
    lines.append("")
    return "\n".join(lines)


def replace_section(text: str, new_section: str) -> str:
    m = SECTION_RE.search(text)
    if not m:
        # append before 变更记录
        for pat in (r"(^##\s*变更记录\s*$)", r"(^##\s*5\.\s*变更记录\s*$)"):
            cm = re.search(pat, text, re.M)
            if cm:
                return text[: cm.start()] + new_section + text[cm.start() :]
        return text.rstrip() + "\n" + new_section
    start = m.start()
    rest = text[m.end() :]
    nxt = re.search(r"^##\s+", rest, re.M)
    end = m.end() + (nxt.start() if nxt else len(rest))
    return text[:start] + new_section + text[end:]


def should_process(path: Path) -> bool:
    name = path.name
    if name.endswith(".test-intent.md") or name.endswith(".test.intent.md"):
        return False
    if not name.endswith(".intent.md"):
        return False
    if "00_索引" in path.parts or "intent_template" in name:
        return False
    return True


def main() -> int:
    ap = argparse.ArgumentParser()
    ap.add_argument("--dry-run", action="store_true")
    args = ap.parse_args()

    updated = 0
    for root in (ROOT / "docs" / "intents", ROOT / "task2app" / "docs" / "intents"):
        if not root.is_dir():
            continue
        for path in sorted(root.rglob("*.intent.md")):
            if not should_process(path):
                continue
            stem = stem_of(path)
            text = path.read_text(encoding="utf-8")
            title = title_of(text, path)

            if stem in NO_EVENT:
                new_sec = build_no_event_section(title, NO_EVENT[stem])
            elif stem in EVENT_ROWS:
                new_sec = build_event_section(EVENT_ROWS[stem])
            else:
                # frontend default no-event already OK; skip untouched
                continue

            new_text = replace_section(text, new_sec)
            if new_text == text:
                continue
            rel = path.relative_to(ROOT)
            if args.dry_run:
                print(f"WOULD {rel}")
            else:
                path.write_text(new_text, encoding="utf-8")
                print(f"UPDATED {rel}")
            updated += 1

    print(f"\nrefined={updated}")
    return 0


if __name__ == "__main__":
    sys.exit(main())
