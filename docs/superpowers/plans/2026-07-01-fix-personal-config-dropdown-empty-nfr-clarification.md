# NFR 澄清: 修复个人配置下拉为空 (Bug Fix)

> 输入:
> - 设计文档: `docs/designs/fix-personal-config-dropdown-empty.md`
> - 价值流文档: `docs/superpowers/plans/2026-07-01-fix-personal-config-dropdown-empty-value-stream.md`
>
> 输出使用者: `/6-ddd-领域设计驱动`, `/7-plans-实施计划`, `/8-build-构建`

## NFR 概览表

| 类别 | 等级 | 理由 |
|------|------|------|
| 全部 | L0 - 不需要 | 纯前端 Bug Fix，无新 NFR 要求 |

## 跳过声明

本次修复为纯前端 Bug Fix，满足 NFR 步骤的所有跳过条件：

1. **不引入新的数据流或外部依赖** — 修复仅限于更正 `ServerConfig.logic.vue` 中一行 JSON 字段名 (`data.results` → `data.configs`)，以及补充 `initFeatureParamsSource()` 中缺失的 `fetchPersonalConfigs()` 调用。无后端变更，无 API 变更，无数据模型变更。
2. **无新增 API** — 使用的 `GET /api/personal/feature-params-configs/` 端点已存在且行为正确。
3. **无特殊 NFR 要求** — 修复前后的行为差异仅限于：下拉从「始终为空」变为「正确显示已创建的配置」。不涉及性能、安全、可用性、一致性等非功能维度变化。

## 领域模型影响

无。本次修复不涉及领域层。

## 权衡与边界

无。本次修复无架构决策。

---

*本 NFR 澄清文档为跳过声明。修复可直接进入实施计划阶段。*
