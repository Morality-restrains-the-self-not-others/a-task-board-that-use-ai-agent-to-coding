# Implementation Plan: 任务 LLM 预算限额

> 设计：`docs/superpowers/specs/2026-05-30-task-llm-budget-limit-design.md`
> 价值流：`docs/superpowers/plans/2026-05-30-task-llm-budget-limit-value-stream.md`

## Increment 1 — 租户开关 + sub-token eligibility

- [x] Migration `0040_llm_budget_governance`（`llm_budget_enabled`）
- [x] Domain `llm_budget_eligibility_service`
- [x] `manage_feature_params` 读写 `llm_budget_enabled`
- [x] task-panel 开关 UI（已有）
- [x] `tests/test_llm_budget_feature_toggle.py`
- [x] `tests/test_llm_budget_sub_token_eligibility.py`

## Increment 2 — 工作空间默认预算

- [x] `WorkspaceModelBudgetDefault` 模型
- [x] `workspace_model_budget_defaults` GET/PATCH
- [x] `tests/test_workspace_model_budget_defaults.py`
- [x] task-panel 预算模态 UI（前端）

## Increment 3 — 任务级预算

- [x] 任务创建 seed `TaskModelBudget`
- [x] 任务预算 GET/PATCH API
- [x] 任务详情预算面板
- [x] `tests/test_task_model_budget_override.py`

## Increment 4 — 用量与 runtime

- [x] 容器 usage POST + 幂等
- [x] `TASK_LLM_BUDGET_POLICY` env 下发
- [x] trae-agent 本地预检
- [x] `tests/test_task_model_budget_usage_api.py`

## Increment 5 — 上调权限

- [x] `TenantBudgetPermission` API
- [x] people/manage UI
- [x] raise endpoint
- [x] `tests/test_tenant_budget_permission.py`

## 验证命令

```bash
cd task2app/Saas_project
python manage.py migrate projects 0040
pytest tests/test_llm_budget_feature_toggle.py \
  tests/test_llm_budget_sub_token_eligibility.py \
  tests/test_workspace_model_budget_defaults.py \
  tests/domain/test_task_llm_budget_domain.py -v
```
