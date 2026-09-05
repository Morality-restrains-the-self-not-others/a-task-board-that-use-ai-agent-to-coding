# 控制台限额 CRUD → Cloud admin API

## 意图

控制台 LLM 限额（工作空间默认 / 任务级预算 / usage 展示）经 Django 用户 API 鉴权后，**读写一律**走 taskCloudService internal admin API；`task_budget.db` 仅 Cloud 打开。

## 已落地（验收）

| 控制台能力 | Cloud API |
|------------|-----------|
| 工作空间默认 GET/PATCH | `GET/POST .../budget/workspace-defaults/`、`.../upsert/` |
| 任务预算 GET/PATCH/raise | `GET/POST .../budget/task-budgets/`、`.../upsert/` |
| 用量展示 | `GET .../budget/task-usage/` |
| 上调权限 | 见 `tenant_budget_permission_cloud.intent.md`（用户态 + admin） |

- Django：`budget_cloud_config_client.py`；视图鉴权/资格校验仍在 Django（defaults/budgets）
- 测试：conftest stub + `budget_cloud_config_stub.py`

## 边界

- 前端 budget-permissions 经网关打 Cloud 用户态 API；不可直打 Cloud internal
- 禁止 Django 打开 `task_budget.db`

## 变更日期

2026-07-14（Permission 切流见同日 `tenant_budget_permission_cloud.intent.md`）


## 业务意图 → 事件对照

> 精修（2026-07-15）：对照 `.ai/08_prompt_management/01_intent_driven_development.md`。

**无对应事件**：控制台 API 迁 Cloud（HTTP），无新增业务事件。

| 业务意图 | 事件名（过去式） | MQ类型/契约 | 发布点 | 消费者/副作用 | 例外理由 |
|---------|----------------|------------|--------|--------------|---------|
| 控制台限额 CRUD → Cloud admin API | — | — | — | — | 控制台 API 迁 Cloud（HTTP），无新增业务事件 |
