# DDD 轻量建模：上层交付物展示

- **日期**: 2026-07-14
- **结论**: 无新聚合/仓储；复用 Task 既有父子引用

## Bounded Context

任务协作（taskTaskService + 工作面板前端）

## 模型

| 类型 | 名称 | 说明 |
|------|------|------|
| Entity / Aggregate Root | Task | `id`, `title`, `deliverable_obj_id`, `parent_task` |
| Value / 导航视图 | ParentDeliverableRef | `{ id, title? }` — 前端展示用，非持久化新实体 |
| Domain Event | 无 | 只读展示 |

## 不变量

- 顶层任务 `parent_task` 为空 → 不展示上层交付物
- 上层引用仅用于对齐导航，不改变任务树写路径
