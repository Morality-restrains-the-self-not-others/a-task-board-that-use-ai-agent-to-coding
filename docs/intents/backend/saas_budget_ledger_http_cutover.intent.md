# Intent: saas-backend 切断 task-budget 直连

将 Django 控制台对 Budget 账本的读写从 `DATABASES['task_budget']` / `TaskBudgetRouter` 改为经 `taskCloudService` internal HTTP；owner 仍为 task-cloud-service。

## 验收

1. 生产 settings 无 `DATABASES['task_budget']`、无 `TaskBudgetRouter`
2. Cloud 提供 workspace-defaults / task-budgets / task-usage internal API
3. 控制台 views 经 `budget_cloud_config_client`；`TenantBudgetPermission` 已迁 Cloud（见 `tenant_budget_permission_cloud.intent.md`）
4. `known_cross_service_access` 删除 saas-backend→task-budget
5. 容器 usage 入账路径（record-usage / model-budget-usage）保持可用



## 业务意图 → 事件对照

> 精修（2026-07-15）：对照 `.ai/08_prompt_management/01_intent_driven_development.md`。

**无对应事件**：切断直连的 HTTP cutover，无新增业务事件。

| 业务意图 | 事件名（过去式） | MQ类型/契约 | 发布点 | 消费者/副作用 | 例外理由 |
|---------|----------------|------------|--------|--------------|---------|
| Intent: saas-backend 切断 task-budget 直连 | — | — | — | — | 切断直连的 HTTP cutover，无新增业务事件 |
## 变更记录

- 2026-07-13：落地 HTTP 客户端 + Cloud CRUD/list API；移出 known-debt
- 2026-07-14：TenantBudgetPermission → task_budget.db + 用户态 API
