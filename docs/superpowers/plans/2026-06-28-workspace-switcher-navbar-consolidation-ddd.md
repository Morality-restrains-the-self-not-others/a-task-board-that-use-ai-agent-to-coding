# DDD 领域建模: Workspace Switcher Navbar Consolidation

> 输入:
> - 设计文档: `docs/plans/2026-06-28-workspace-switcher-navbar-consolidation.md`
> - 价值流文档: `docs/superpowers/plans/2026-06-28-workspace-switcher-navbar-consolidation-value-stream.md`
> - NFR 澄清: `docs/superpowers/plans/2026-06-28-workspace-switcher-navbar-consolidation-nfr-clarification.md`
>
> 输出使用者: `/6-plans-实施计划`

## 跳过声明

本次变更为**纯前端组件位置迁移**，不引入新领域概念。理由：

- ❌ 无新实体、值对象、聚合或领域服务
- ❌ 无新仓储接口或领域事件
- ❌ 无新限界上下文
- ❌ 后端 API 零变化

现有领域模型不变：
- **Workspace 实体**（`projects_workspace`）— 不变
- **Workspace Switch 领域服务**（`POST /api/tenant/{tid}/projects/switch/`）— 不变
- **项目与工作空间限界上下文** — 不变

唯一变化：Workspace Switch 的 **UX 触发点**从 `WorkPanelHeader` 迁至 `Navbar`，属于展示层调整。

## 自检

- [x] 无新领域文件需要生成
- [x] 现有领域模型不受影响
- [x] 跳过理由已文档化
