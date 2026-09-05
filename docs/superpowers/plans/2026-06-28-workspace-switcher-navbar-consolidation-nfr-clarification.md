# NFR 澄清: Workspace Switcher Navbar Consolidation

> 输入:
> - 设计文档: `docs/plans/2026-06-28-workspace-switcher-navbar-consolidation.md`
> - 价值流文档: `docs/superpowers/plans/2026-06-28-workspace-switcher-navbar-consolidation-value-stream.md`
>
> 输出使用者: `/6-plans-实施计划`（跳过 DDD，无新领域概念）

## 跳过声明

本次变更为**纯前端组件位置迁移**——将 `WorkspaceSwitcher` 从 `WorkPanelHeader.vue` 移至 `Navbar.ui.vue`，不涉及：

- ❌ 新 API / 新后端逻辑
- ❌ 新数据流或外部依赖
- ❌ 数据库 schema 变更
- ❌ 新领域概念或业务规则
- ❌ 认证/授权逻辑变更
- ❌ 性能特征变更（同一组件、同一 API 调用、同一全页刷新行为）

**NFR 判定：所有类别适用「不适用 — 纯 UI 组件迁移，无新增质量需求」。**

变更前后的用户可感知行为完全一致：
1. 拉取工作空间列表 → 同一 API (`GET /api/tenant/{tid}/workspaces/`)
2. 切换工作空间 → 同一 API (`POST /api/tenant/{tid}/projects/switch/`)
3. 全页刷新 → 同一行为 (`window.location.href`)

唯一差异：触发控件从页面 Header 移至全局 Navbar，UX 更一致。

## 质量场景

无新增质量场景。现有 `switch-workspace` 的后端测试 (`projects/view_test/WorkspaceViewSet_switch_workspace_test.py`) 继续覆盖后端行为。

## 领域模型影响

无。不新增、不修改任何领域概念。

## 权衡与边界

无新增权衡。变更仅影响前端渲染位置，行为零改变。
