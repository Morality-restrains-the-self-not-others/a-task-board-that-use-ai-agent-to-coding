# Value Stream: 任务 LLM 预算限额

> 设计：`docs/superpowers/specs/2026-05-30-task-llm-budget-limit-design.md`

## Related Value Streams

- **tenant-feature-params**（`value-stream.yaml`）：**扩展** — `llm_budget_enabled` + `providers[].use_sub_token` 作为预算 eligibility
- **task-management**：**扩展** — 任务创建时 seed `TaskModelBudget`（仅 sub-token provider）
- **task-detail-runtime-relay**：**扩展** — 条件下发 `TASK_LLM_BUDGET_POLICY`、容器 usage 上报

Greenfield stream：`task-llm-budget-governance`（domain: 任务协作）

## Value Summary

租户管理员 opt-in 开启 LLM 预算限额后，对**已启用派生子 Key** 的大模型接入按端点×模型设置 CNY 默认预算；任务继承并可覆盖；超额硬拦截，授权成员可临时上调。

## End-to-End Flow

```text
[功能参数页勾选 use_sub_token]
  → [task-panel 开启 llm_budget_enabled]
  → [配置工作空间默认单价+预算]
  → [创建任务 → seed TaskModelBudget]
  → [容器 bootstrap → TASK_LLM_BUDGET_POLICY]
  → [agent LLM 调用前检查 spent < limit]
  → [超额 → 硬拦截 + 任务详情 banner]
  → [授权用户临时上调 / 任务 Owner 覆盖预算]
```

## Value Increments

### Increment 1: 租户开关 + sub-token eligibility（Thin Slice）

**Value to user：** 租户可开关功能；仅 sub-token provider 可进入预算 API。

**Scope：**

- Migration：`llm_budget_enabled`
- `is_budget_eligible_provider()` 领域逻辑
- `require_llm_budget_enabled` decorator
- `GET/PATCH feature-params` 含开关
- task-panel 开关 UI
- `tests/test_llm_budget_feature_toggle.py`
- `tests/test_llm_budget_sub_token_eligibility.py`

**Depends on：** 无

### Increment 2: 工作空间默认预算 CRUD

**Value to user：** 管理员为 sub-token 模型配置单价与默认预算上限（元）。

**Scope：**

- `WorkspaceModelBudgetDefault` 模型 + migration
- `GET/PATCH .../model-budget-defaults/`
- task-panel 预算模态 UI
- `tests/test_workspace_model_budget_defaults.py`

**Depends on：** Increment 1

### Increment 3: 任务级预算继承与覆盖

**Value to user：** 新任务继承默认；任务详情可修改单模型预算。

**Scope：**

- `TaskModelBudget` 模型 + 任务创建 hook
- 任务预算 GET/PATCH API
- 任务详情「LLM 预算」面板（仅 sub-token 模型）
- `tests/test_task_model_budget_override.py`

**Depends on：** Increment 2

### Increment 4: 用量上报 + 运行时 policy 下发

**Value to user：** 容器收到 budget policy；平台累计 spent；agent 可拦截。

**Scope：**

- `TaskModelBudgetUsage` 模型
- 容器 usage POST API（AccessToken 鉴权）
- `fetch_tenant_feature_params_env` 或独立 endpoint 附加 `TASK_LLM_BUDGET_POLICY`
- trae-agent 本地预检（最小）
- `tests/test_task_model_budget_usage_api.py`

**Depends on：** Increment 3

### Increment 5: 上调权限 + 临时 raise

**Value to user：** people/manage 配置权限；超额后授权用户二次确认上调。

**Scope：**

- `TenantBudgetPermission` 模型 + API
- `POST .../model-budgets/raise/`
- people/manage + 任务详情「临时上调」
- `tests/test_tenant_budget_permission.py`

**Depends on：** Increment 3

## 验收标准

| 增量 | 验收 |
|------|------|
| 1 | 默认关闭；开启后 sub-token 校验通过；master provider 403 |
| 2 | 工作空间 CRUD；非 eligible provider 403 |
| 3 | 任务创建继承；详情 PATCH 覆盖 |
| 4 | usage 幂等累计；env 含 policy |
| 5 | 无权限 raise 403；有权限上调成功 |
