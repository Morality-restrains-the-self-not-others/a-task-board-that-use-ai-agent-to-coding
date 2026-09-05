# 删除 Django budget/record-usage-batch thin

## 意图

删除 Django `budget/record-usage-batch` HTTP 壳；容器与 AI 写账本一律直达 `taskCloudService` `/api/internal/budget/*`。

## 验收

1. `/api/internal/cloud/budget/record-usage-batch/` → **404**
2. `budget_cloud_client.record_usage_batch` 已删除；`commit_usage` / `reserve_or_deny` 仅供 Django 控制台路径（`task_llm_budget_service`）
3. **taskAIEndPoint** Budget 已直打 Cloud（见 `ai_endpoint_budget_direct_cloud.intent.md`）；Django `task-ai-endpoint/budget/*` 已删
4. 相关 Django 测试断言 404；Go Budget 单测仍绿

## 变更日期

2026-07-10



## 业务意图 → 事件对照

> 精修（2026-07-15）：对照 `.ai/08_prompt_management/01_intent_driven_development.md`。

**无对应事件**：删除 Django thin 壳，无新增业务事件投递。

| 业务意图 | 事件名（过去式） | MQ类型/契约 | 发布点 | 消费者/副作用 | 例外理由 |
|---------|----------------|------------|--------|--------------|---------|
| 删除 Django budget/record-usage-batch thin | — | — | — | — | 删除 Django thin 壳，无新增业务事件投递 |
## 变更记录

- 2026-07-13：交叉引用 AI endpoint Budget 直打 Cloud 完成
