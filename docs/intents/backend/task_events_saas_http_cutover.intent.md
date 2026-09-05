# Intent: task-events 切断 saas 直连

将 `taskEvents` 对各 saas 表的生产直连（`saas.Open` / `ResolveDatabasePath("saas")`）改为经 **saas-backend** internal intent HTTP；工作空间行写入经 Django 编排转发 **taskProjectService**（表已迁出 saas）。

## Python 例外（Go-first 门禁）

| 项 | 说明 |
|---|---|
| 落点 | 扩展现有 `saas-backend` `/api/internal/task-events/`（django-internal） |
| 理由 | saas 库 owner 为 saas-backend；属「Django 作为已声明的 internal 真源」例外；非公网入口 |
| 迁表 | **不**迁 `cloud_cloudserverevent` / `cloud_accesskeyiamidassociation`（仍属 saas）；`workspaces` / `workspace_accesses` 已在 task-project。再迁评估与验收门禁见 [`cloud_domain_tables_task_cloud_migration.intent.md`](./cloud_domain_tables_task_cloud_migration.intent.md) |
| 风险缓解 | internal secret；幂等语义与现 Go sqlite Repository 对齐；handler 测 httptest |

## 验收

1. 各 `taskEvents/cmd/**/main.go` 无 `saas.Open`
2. 生产代码无 `ResolveDatabasePath("saas")`
3. `known_cross_service_access` 删除 task-events→saas
4. ownership checker 无该项 WARN
5. 相关 go test 通过

## API 清单（saas-backend）

前缀：`/api/internal/task-events/intents/`

| Method | Path | 对应原 Repository |
|---|---|---|
| GET | `company-by-creator/?user_id=` | CompanyByCreator |
| POST | `create-company/` | CreateCompanyForUser |
| POST | `upsert-user-profile/` | UpsertUserProfileUsername |
| POST | `set-default-deliverable/` | IntentSetDefaultDeliverable |
| POST | `set-default-progress/` | IntentSetDefaultProgress |
| POST | `create-default-workspace/` | IntentCreateDefaultWorkspace |
| POST | `handle-workspace-created/` | HandleWorkspaceCreated |
| POST | `create-access-key-iam/` | CreateAccessKeyIAMAssociation |
| GET | `latest-pending-start-event/?company_id=&task_id=` | LatestPendingStartEvent |
| POST | `update-cloud-server-event-status/` | UpdateCloudServerEventStatus |
| POST | `update-cloud-server-event-data/` | UpdateCloudServerEventData |

`CloudAuthorizationByID` 仍直打 taskCloudService（勿回退）。



## 业务意图 → 事件对照

> 精修（2026-07-15）：对照 `.ai/08_prompt_management/01_intent_driven_development.md`。

**无对应事件**：切断 saas 直连的 cutover，无新增业务事件。

| 业务意图 | 事件名（过去式） | MQ类型/契约 | 发布点 | 消费者/副作用 | 例外理由 |
|---------|----------------|------------|--------|--------------|---------|
| Intent: task-events 切断 saas 直连 | — | — | — | — | 切断 saas 直连的 cutover，无新增业务事件 |
## 变更记录

- 2026-07-14：落地 intent API + taskEvents HTTP client；移出 known-debt
