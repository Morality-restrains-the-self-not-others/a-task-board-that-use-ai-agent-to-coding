# Plan: LLM 预算端点级启用（方案 A）

> Design: `docs/superpowers/specs/2026-05-30-llm-budget-endpoint-level-design.md`

## Tasks

- [x] **T1** `providers[].budget_enabled` 规范化 + feature-params POST 拒绝 `llm_budget_enabled`
- [x] **T2** eligibility：`use_sub_token ∧ budget_enabled`
- [x] **T3** 数据迁移 `0041_provider_budget_enabled`
- [x] **T4** 功能参数页：端点级「启用 LLM 预算」；移除租户总开关卡片
- [x] **T5** 测试更新 + value-stream 字段

## Verify

```bash
cd task2app/Saas_project && pytest tests/test_llm_budget_endpoint_enable.py tests/test_llm_budget_feature_toggle.py tests/test_llm_budget_sub_token_eligibility.py tests/domain/test_task_llm_budget_domain.py -q
cd task2app/front_project/app && npm run build
```

## Ship

- Commit: `c782e2b2` on `feat/task-llm-budget-limit`（已 push）
- PR：需本地 `gh auth login` 后执行 `gh pr create`
