# TenantBudgetPermission 统一到 Cloud

## 意图

将 `TenantBudgetPermission`（预算上调权限）从 Django ORM / saas 表迁至 **task-cloud-service**，并提供：

1. **Internal admin API**（服务间，`X-Internal-Secret`）
2. **用户态 API**（浏览器经 APISIX forward-auth，**禁止**复用 internal secret）

## 已落地

| 能力 | 路径 |
|------|------|
| 权限 list | `GET /api/internal/budget/tenant-permissions/?company_id=` |
| 权限 upsert | `POST /api/internal/budget/tenant-permissions/upsert/` |
| 上调判定 | `POST /api/internal/budget/tenant-permissions/evaluate-raise/` |
| 用户态 GET/PATCH | `/api/tenant/{tid}/budget-permissions/` → Cloud（网关 priority 870） |

- 存储：`task_budget.db` 表 `projects_tenant_budget_permission`（Cloud 独占）
- 成员/小组校验：仍经 Django internal（`resolve-user-member` / `user-in-group`）
- Django：`budget_cloud_config_client` + 薄代理视图（单测兜底）；`TenantBudgetPermission` `managed=False`
- 前端路径不变，靠网关切流
- **saas 旧表**：本地确认 0 行后已 DROP（`drop_saas_tenant_budget_permission.sh`）；其他环境先迁再跑脚本

## 边界

- 前端不可带 `X-Internal-Secret` 直打 Cloud internal
- model-budget-defaults / model-budgets 用户态见 `budget_console_user_api_cloud.intent.md`

## 变更日期

2026-07-14


## 业务意图 → 事件对照

> 精修（2026-07-15）：对照 `.ai/08_prompt_management/01_intent_driven_development.md`。

**无对应事件**：权限数据迁 Cloud（HTTP/存储），无新增业务事件。

| 业务意图 | 事件名（过去式） | MQ类型/契约 | 发布点 | 消费者/副作用 | 例外理由 |
|---------|----------------|------------|--------|--------------|---------|
| TenantBudgetPermission 统一到 Cloud | — | — | — | — | 权限数据迁 Cloud（HTTP/存储），无新增业务事件 |
