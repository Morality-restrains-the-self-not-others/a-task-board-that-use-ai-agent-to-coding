# DDD Model: 任务 LLM 预算限额

> 来源：`docs/superpowers/specs/2026-05-30-task-llm-budget-limit-design.md`
> NFR：`docs/superpowers/plans/2026-05-30-task-llm-budget-limit-nfr-clarification.md`

## 1) 限界上下文

- **TaskLlmBudget**：工作空间默认、任务预算、用量账本、上调权限
- **TenantLlmConfig**（已有）：`providers[].use_sub_token` 作为 eligibility 输入

## 2) 值对象

- `ModelEndpointKey(provider, base_url, model_name)` — `projects/domain/task_llm_budget/value_objects/`
- `CnyAmount` — 以 Decimal 元表示，无 currency 维度（实现于 services）

## 3) 领域服务

- `llm_budget_eligibility_service` — `is_budget_eligible_provider_entry`, `expand_budget_model_keys`
- `budget_cost_calculator` — `calculate_spent_cny`

## 4) 实体（持久化 ORM，应用层映射）

- `WorkspaceModelBudgetDefault`
- `TaskModelBudget`
- `TaskModelBudgetUsage`
- `TenantBudgetPermission`

## 5) 领域事件（planned）

- `TaskModelBudgetExhausted`
- `TaskModelBudgetRaised`
- `TaskModelBudgetUsageReported`

## 6) Hard Gate

- [x] eligibility 在 domain 层，无 ORM 导入
- [x] 费用公式在 domain 层
- [ ] 完整聚合 `TaskBudgetPolicy`（Increment 3–4 补充）
- [ ] 仓储 ABC（Increment 3–4 补充）
