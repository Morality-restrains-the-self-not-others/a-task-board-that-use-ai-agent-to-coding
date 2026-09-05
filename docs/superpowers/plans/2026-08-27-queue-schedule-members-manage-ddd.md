# 自动调度安排 · 排队任务卡可管理 — 领域建模

- 日期：2026-08-27

## 结论

**不新增领域层文件。** 本增量是既有聚合的新 UI 入口。

## 复用聚合

- **QueuedAutoRunMembership**（既有）：`task_id` PK。加入/离开仍由 taskTaskService `enqueueQueuedAutoRun` / `dequeueQueuedAutoRun` 执行。
- **WorkspaceScheduleRhythm**（既有）：仅读 `enabled` 作前端门闩。

## 领域事件（不新增）

| 业务意图 | 事件 | 发布点 | 例外 |
|----------|------|--------|------|
| 从排队卡加入队列 | TaskQueuedForAutoRun | taskTaskService PATCH → enqueue | 沿用既有；证据豁免见 B-055 |
| 从排队卡离开队列 | TaskDequeuedFromAutoRun | taskTaskService PATCH → dequeue | 同上 |
| 搜索候选任务 | — | 纯查询 | 无服务端状态变更 |
| 未启用阻断入队 | — | 纯前端，未发 PATCH | 无事件 |

## 不变式

1. 未启用工作空间节奏时，本卡不得发出入队 PATCH（与任务详情门闩一致）。
2. 出队须用户确认；取消不得 PATCH。
3. 成员列表以 GET queue-schedule 快照为读模型，写成功后必须刷新。
