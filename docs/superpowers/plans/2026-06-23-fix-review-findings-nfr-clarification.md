# NFR 澄清: Fix Code Review Findings (runAll)

> 输入:
> - 设计文档: `docs/superpowers/specs/2026-06-23-fix-review-findings-design.md`
> - 价值流文档: `docs/superpowers/plans/2026-06-23-fix-review-findings-value-stream.md`
>
> 输出使用者: `/5-ddd-领域设计驱动`, `/6-plans-实施计划`, `/7-build-构建`

## NFR 概览表

| 类别 | 等级 | 理由 |
|------|------|------|
| 全部 | L0 - 跳过 | 纯 bug 修复，无新功能、无新数据流、无新外部依赖 |

## 跳过声明

本次变更为 3 个已有功能的缺陷修复：
1. `handleBuildAllAction` 异步化 — 遵循已有的 `handleStopAllAction` 异步模式
2. `PlanStopAll` 空列表 — 边界条件处理
3. `collapseAllGroups` 空数据 — JS 边界条件处理

不引入新的用户流程、数据模型、外部依赖或性能特性。所有 NFR 类别均跳过。
