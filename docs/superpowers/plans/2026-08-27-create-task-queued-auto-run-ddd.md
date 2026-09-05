# DDD — 创建任务可选加入自动调度队列

- 日期：2026-08-27
- 限界上下文：taskTaskService（任务 + 排队调度）；taskFE（创建意图 UI）

## 聚合 / 值对象

- **Task**：`auto_run`（是否应按运行模版自动跑）、`queued_auto_run`（是否为工作空间队列成员）。
- **WorkspaceScheduleRhythm**：`enabled` 决定创建 UI 是否提供入队选择。
- **QueuedMembership**：入队后的调度状态（queued / deferred / starting）。

## 领域规则

1. 入队不蕴含立即启服；立即启服不蕴含入队。
2. 同一次创建若两者皆真：入队优先于立即 start-vm（避免双机）。
3. `force_auto_run` 覆盖推迟，仍立即启服。

## 端口

- 既有：enqueue / dequeue / GET queue-schedule / auto_run start-vm。
- 本增量不新增端口；应用服务在 create/update 编排里增加「queued ⇒ 跳过 scheduleTaskAutoRun」。

## 事件

沿用 `TaskQueuedForAutoRun`、`QueuedAutoRunStarted`。推迟启服打结构化日志 `task_auto_run_deferred_to_queue`，不新增事件类型。
