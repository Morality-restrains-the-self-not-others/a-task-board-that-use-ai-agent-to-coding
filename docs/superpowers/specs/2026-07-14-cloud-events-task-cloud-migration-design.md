# Design: 门禁 1–3 + 刀 A/B — cloud_server_events / IAM 迁入 taskCloudService

> 日期: 2026-07-14  
> 状态: approved (goal-mode auto)  
> Intent: `docs/intents/backend/cloud_domain_tables_task_cloud_migration.intent.md`

## 目标

1. **门禁 1**：`finalizeStartVmInGo` 本地持久化 `cloud_server_events`，不再调用 Django `persist-start-vm-auto` / `delete-start-vm-event`；容器令牌改由 Go 直调 task-credential `/v1/token/init`。
2. **门禁 2**：Django `POST /api/cloud/start-server/` 返回 410（与 start-vm 一致）。
3. **门禁 3**：`server-startup-status` 由 taskCloudService 读本地 events + `cloud_server_configs`；不再 proxy Django。SSE 生产路径已是 Kafka→taskSSE，保持不变。
4. **刀 A**：`access_key_iam_associations` 落 task_cloud；taskEvents 改打 Cloud internal；数据迁移 + 删 Django ORM。
5. **刀 B**：taskEvents 事件 status/data/latest 改打 Cloud；saas 行迁入；删 Django `CloudServerEvent`；更新 ownership。

## 数据模型（task_cloud.db）

- `cloud_server_events`：id(TEXT snowflake)、company_id、workspace_id、task_id、… 无 FK 到 accounts_company
- `access_key_iam_associations`：id、cloud_platform_auth_id、access_key、iam_id

## Internal API（taskCloudService）

| Method | Path |
|---|---|
| GET | `/api/internal/cloud-server-events/latest-pending-start` |
| POST | `/api/internal/cloud-server-events/update-status` |
| POST | `/api/internal/cloud-server-events/update-data` |
| POST | `/api/internal/access-key-iam-associations/` |

鉴权：`X-Internal-Secret`（与现有 internal 一致）。

## 迁移窗口

- `db/task_cloud/migrate_cloud_events_from_saas.sh`：显式列名 `INSERT OR IGNORE`，保留 Snowflake id
- Django migration：drop ORM tables after copy
- `db/table_ownership.yaml`：两表 owner → task-cloud-service

## 非目标

- 不改前端 SSE URL
- 不改 taskBill `cloud_event_id` 幂等语义（ID 原样）
- 本轮不强制 ArchiMate 全套视图（ownership 文档 + intent 更新）

## 交付状态（2026-07-14）

门禁 1–5 与刀 A/B 已落地；见 intent `cloud_domain_tables_task_cloud_migration.intent.md`（status: done）。

## 验收

- Go test：finalize 无 Django persist；status poll 本地；IAM create
- ownership checker：0 known-debt；两表在 task-cloud
- live smoke 仍绿
- Django `cloud.0052` 已可在本机应用并 DROP saas 旧表
