# cloud_cloudserverevent / IAM 关联表迁入 taskCloudService — 决策与再迁验收

> 状态: **done** | 日期: 2026-07-14  
> 关联：[`task_events_saas_http_cutover.intent.md`](./task_events_saas_http_cutover.intent.md)、[`docs/architecture/table-to-owner.md`](../../architecture/table-to-owner.md)、[`cloud_domain_tables_task_cloud_DEPLOY.md`](./cloud_domain_tables_task_cloud_DEPLOY.md)  
> 架构视图：`docs/architecture/v21-application-integration-20260714-0130-claude.{puml,archimate,mermaid.md}`  
> 规范：`.ai/01_project_constraints/19_single_service_data_ownership.md`

## 决策

**已完成**将 `cloud_cloudserverevent` 与 `cloud_accesskeyiamidassociation` 从 saas 迁入 `task_cloud.db`（刀 A + 刀 B）。

## 门禁（2026-07-14 切流）

| # | 条件 | 状态 |
|---|---|---|
| 1 | `finalizeStartVmInGo` 本地持久化 `cloud_server_events`，不再调用 Django persist/delete | ✅ |
| 2 | 旧 Django `POST /api/cloud/start-server/` 返回 410 | ✅ |
| 3 | `server-startup-status` 由 taskCloudService 提供；Django ViewSet 返回 410 | ✅ |
| 4 | `cloud_event_id` 计费幂等 ID 迁表后保持不变（Snowflake 原样搬迁） | ✅ |
| 5 | saas → task_cloud 数据迁移（HTTP import + shell 脚本） | ✅ |

## 现状（合规）

| 表 | 物理库 | owner | 跨服务访问方式 |
|---|---|---|---|
| `cloud_server_events` | task_cloud.db | task-cloud-service | taskEvents 经 Cloud internal API |
| `access_key_iam_associations` | task_cloud.db | task-cloud-service | taskEvents 经 Cloud internal API |
| `cloud_platform_authorizations` 等 | task_cloud.db | task-cloud-service | 已在 Go |

## 实现摘要

- **taskEvents**：`CreateAccessKeyIAMAssociation` / `LatestPendingStartEvent` / `UpdateCloudServerEvent*` → `taskCloudService` internal HTTP（`doCloudJSON`）
- **taskCloudService**：batch import `POST .../cloud-server-events/import`、`POST .../access-key-iam-associations/import`
- **Django**：`cloud.0052` RunPython import + DROP；intent 410；`cloud_start_server_legacy` / `server-startup-status` ViewSet 410
- **脚本**：`db/task_cloud/migrate_cloud_events_from_saas.sh`
- **ownership**：`db/table_ownership.yaml` 两表 owner → `task-cloud-service`



## 业务意图 → 事件对照

> 精修（2026-07-15）：对照 `.ai/08_prompt_management/01_intent_driven_development.md`。

**无对应事件**：表归属迁移决策/验收，无新增业务事件。

| 业务意图 | 事件名（过去式） | MQ类型/契约 | 发布点 | 消费者/副作用 | 例外理由 |
|---------|----------------|------------|--------|--------------|---------|
| cloud_cloudserverevent / IAM 关联表迁入 taskCloudService — 决策与再迁验收 | — | — | — | — | 表归属迁移决策/验收，无新增业务事件 |
## 变更记录

| 日期 | 说明 |
|---|---|
| 2026-07-14 | Archi `--loadModel` v21 OK；本机按 DEPLOY 清单全项勾选（证据 `_deploy_evidence_20260714/`） |
| 2026-07-14 | 清理 django_client trace 残留；部署清单 DEPLOY.md；ArchiMate v21 三类伴生 |
| 2026-07-14 | 门禁 1–5 ✅；刀 A/B 完成；intent 状态 → done |
| 2026-07-14 | 按门禁表复核：1–3 未满足；Slice A/B 继续 deferred 并写明 A 可独立但无紧迫性 |
| 2026-07-14 | 评估结论：暂缓；记录再迁验收标准 |
