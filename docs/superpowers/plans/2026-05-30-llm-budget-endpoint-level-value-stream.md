# Value Stream: LLM 预算端点级启用

**设计：** `docs/superpowers/specs/2026-05-30-llm-budget-endpoint-level-design.md`

## Related Value Streams

- **task-llm-budget-governance**（修改）：租户总开关 → `providers[].budget_enabled`
- **tenant-feature-params**（依赖）：功能参数 CRUD 承载端点配置

## Increments

| # | 增量 | 验收 |
|---|------|------|
| 1 | 后端 `budget_enabled` + eligibility | `test_llm_budget_endpoint_enable.py` 绿 |
| 2 | 迁移 + 聚合 `llm_budget_enabled` | 旧租户行为不变 |
| 3 | 功能参数 UI 端点级勾选 | 构建通过；无租户总开关卡片 |
