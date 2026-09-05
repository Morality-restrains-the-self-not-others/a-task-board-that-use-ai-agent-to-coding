# DDD 领域建模: 修复个人配置下拉为空 (Bug Fix)

> 输入:
> - 设计文档: `docs/designs/fix-personal-config-dropdown-empty.md`
> - 价值流文档: `docs/superpowers/plans/2026-07-01-fix-personal-config-dropdown-empty-value-stream.md`
> - NFR 澄清: `docs/superpowers/plans/2026-07-01-fix-personal-config-dropdown-empty-nfr-clarification.md`
>
> 输出使用者: `/7-plans-实施计划`

## 跳过声明

本次修复为纯前端 Bug Fix，满足 DDD 步骤的跳过条件：

1. **纯前端改动** — 修复仅限于 `ServerConfig.logic.vue` 中一行 JSON 字段名更正 (`data.results` → `data.configs`) 及 `initFeatureParamsSource()` 中补充 `fetchPersonalConfigs()` 调用。
2. **不涉及新的业务概念** — 所有领域概念（`PersonalFeatureParamsConfig`、`FeatureParamsSource`、`FeatureParamsResolver`）已在前序 `feature-params-hierarchy` 流中完成领域建模。
3. **无新增实体/值对象/聚合/领域事件** — 本次修复不引入任何新的领域模型元素。
4. **无新增端口接口** — 使用的 API 端点 `GET /api/personal/feature-params-configs/` 已存在且行为正确。

## 已有领域模型参考

本次修复涉及的领域模型已在 `task2app/Saas_project/projects/domain/feature_params/` 中完整定义：

- **实体**: `PersonalFeatureParamsConfig`, `TenantFeatureParams`, `WorkspaceFeatureParams`
- **值对象**: `FeatureParamsSource` (company/workspace/personal)
- **领域服务**: `FeatureParamsResolver` (三级回退解析)
- **端口**: `PersonalFeatureParamsConfigRepository`, `WorkspaceGovernancePort`
- **事件**: `FeatureParamsResolved`

以上均无需修改。

---

*本 DDD 文档为跳过声明。修复可直接进入实施计划阶段。*
