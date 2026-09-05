# AI endpoint 三条 Django thin 迁 Go

## 意图

将 taskAIEndPoint 仍依赖的 Django internal API 全部迁出：

| API | 目标 |
|-----|------|
| validate-proxy-token | taskCredentialService `/v1/token/validate` + 本地 scope 比对 |
| resolve-route | taskCloudService `/api/internal/ai-endpoint/resolve-route/` |
| upstream-credentials | taskCloudService `/api/internal/ai-endpoint/upstream-credentials/` |

并收紧控制台：删除 `task_llm_budget_service` 死代码写路径与 `budget_cloud_client` HTTP 壳；控制台仅 ORM 读共享 `task_budget.db`。

## 验收

1. Django `/api/internal/task-ai-endpoint/**` → **404**（urls 已卸）
2. Cloud resolve/credentials 单测绿；AI endpoint validate/route/budget 单测绿
3. conf：`taskCloudService` + `taskCredentialService` URL
4. 控制台 `build_task_budget_items` / 权限函数仍可用；无 Django → Cloud budget HTTP 旁路

## 变更日期

2026-07-13


## 业务意图 → 事件对照

> 精修（2026-07-15）：对照 `.ai/08_prompt_management/01_intent_driven_development.md`。

**无对应事件**：Django thin 迁 Go（HTTP），无新增业务事件。

| 业务意图 | 事件名（过去式） | MQ类型/契约 | 发布点 | 消费者/副作用 | 例外理由 |
|---------|----------------|------------|--------|--------------|---------|
| AI endpoint 三条 Django thin 迁 Go | — | — | — | — | Django thin 迁 Go（HTTP），无新增业务事件 |
