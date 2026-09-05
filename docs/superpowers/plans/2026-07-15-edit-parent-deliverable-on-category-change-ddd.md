# DDD：编辑时上层交付物（轻量）

- **日期**: 2026-07-15
- **结论**: 无新聚合；扩展既有 Task 写路径的不变式

## Bounded Context

Task Collaboration（taskTaskService + 前端 work-panel/task-detail）

## 不变式（强化）

1. Task 的 `deliverable_obj_id` 指向当前工作空间交付物体系中的类别。
2. 若类别非顶层（`order` > min），则 `parent_task_id` 必须指向「上一 order 层」的 Task。
3. 若类别为顶层，则 `parent_task_id` 为空。

## 模型

| 概念 | 类型 | 备注 |
|------|------|------|
| Task | Aggregate Root | 已有 |
| DeliverableCategory | Value（前端视图） | 由 DeliverableObj.order 推导 |
| ParentDeliverableRef | Value | Task.parent_task_id |

无新 repository / 无新 domain event。
