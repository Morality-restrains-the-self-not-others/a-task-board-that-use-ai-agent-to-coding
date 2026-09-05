# AI endpoint Budget 直打 Cloud

## 意图

将 taskAIEndPoint 的 `reserve-or-deny` / `commit-usage` 从 Django thin
（`/api/internal/task-ai-endpoint/budget/*`）改为直打 `taskCloudService`
`/api/internal/budget/*`，去掉 Django 预算转发壳。

## 验收

1. taskAIEndPoint `cloudBudgetGate` / `cloudCommitUsage` → Cloud；带 `X-Internal-Secret`（可空）
2. Django `/api/internal/task-ai-endpoint/budget/reserve-or-deny/` 与 `commit-usage/` → **404**
3. Django 仍保留 validate-proxy-token / resolve-route / upstream-credentials
4. Go 单测覆盖 Cloud budget 调用；Django 测试断言 budget URL 404
5. conf：`conf/ai/task-ai-endpoint/config.yaml` 含 `taskCloudService.url`

## 变更日期

2026-07-13


## 业务意图 → 事件对照

> 精修（2026-07-15）：对照 `.ai/08_prompt_management/01_intent_driven_development.md`。

**无对应事件**：API 路由迁 Go（HTTP），无新增业务事件。

| 业务意图 | 事件名（过去式） | MQ类型/契约 | 发布点 | 消费者/副作用 | 例外理由 |
|---------|----------------|------------|--------|--------------|---------|
| AI endpoint Budget 直打 Cloud | — | — | — | — | API 路由迁 Go（HTTP），无新增业务事件 |
