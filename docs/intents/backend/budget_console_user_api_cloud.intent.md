# 控制台预算用户态 API → Cloud

## 意图

浏览器直打的 LLM 预算控制台 API（非 internal secret）统一到 taskCloudService，经 APISIX forward-auth：

| 路径 | 方法 |
|------|------|
| `/api/tenant/{tid}/budget-permissions/` | GET/PATCH |
| `/api/tenant/{tid}/workspaces/{wid}/model-budget-defaults/` | GET/PATCH |
| `/api/tenant/{tid}/workspace/{wid}/todos/{task}/model-budgets/` | GET/PATCH |
| `.../model-budgets/raise/` | POST |

## 已落地

- Cloud：`budget_user_handlers.go` + `budget_permission_handlers.go`
- 网关：`task-cloud-budget-console` priority 870（高于 task-task-service 862，避免 todos 误路由）
- Django 视图保留作单测兜底，标注 Deprecated
- saas `projects_tenant_budget_permission`：本地确认 0 行后已 DROP；脚本 `db/task_budget/drop_saas_tenant_budget_permission.sh`

## 边界

- 禁止前端带 `X-Internal-Secret`
- 工作空间/任务存在性分别经 Project / Task 服务 HTTP
- 成员与小组校验经 Django internal

## 变更日期

2026-07-14


## 业务意图 → 事件对照

> 精修（2026-07-15）：对照 `.ai/08_prompt_management/01_intent_driven_development.md`。

**无对应事件**：控制台用户 API 迁 Cloud（HTTP），无新增业务事件。

| 业务意图 | 事件名（过去式） | MQ类型/契约 | 发布点 | 消费者/副作用 | 例外理由 |
|---------|----------------|------------|--------|--------------|---------|
| 控制台预算用户态 API → Cloud | — | — | — | — | 控制台用户 API 迁 Cloud（HTTP），无新增业务事件 |
