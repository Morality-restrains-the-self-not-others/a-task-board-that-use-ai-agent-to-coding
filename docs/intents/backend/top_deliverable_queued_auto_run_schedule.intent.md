# 意图：顶层交付物排队自动执行调度节奏

- 日期：2026-07-19
- 设计：`docs/superpowers/specs/2026-07-19-top-deliverable-queued-auto-run-schedule-design.md`
- value-stream step：`top-deliverable-queued-auto-run-schedule`

## 业务意图 → 事件对照

| 业务意图 | 事件名（过去式） | 发布点 | 消费者/副作用 | 例外理由 |
|----------|------------------|--------|---------------|----------|
| 更新顶层调度节奏 | ScheduleRhythmUpdated | taskTaskService `applyScheduleRhythmFromBody` | 前端任务详情节奏面板刷新 | 证据豁免：首期 publishDomainEvent 日志/审计，不强制 Kafka |
| 加入排队 | TaskQueuedForAutoRun | taskTaskService `enqueueQueuedAutoRun` | dispatcher 纳入出队候选 | 证据豁免：首期 publishDomainEvent 日志/审计，不强制 Kafka |
| 离开排队 | TaskDequeuedFromAutoRun | taskTaskService `dequeueQueuedAutoRun` | 释放排队占用/UI 出队 | 证据豁免：首期 publishDomainEvent 日志/审计，不强制 Kafka |
| 窗外/未启用 defer | QueuedAutoRunDeferred | taskTaskService `dispatchTopQueue` | UI 展示 deferred 文案 | 证据豁免：首期 publishDomainEvent 日志/审计，不强制 Kafka |
| 调度启服 | QueuedAutoRunStarted | taskTaskService `startQueuedMembership` | Cloud `started_via=queued_schedule` | 证据豁免：首期 publishDomainEvent 日志/审计，不强制 Kafka |


> 窗口自动关闭扩展见 `schedule_rhythm_auto_close.intent.md`（2026-07-22 增量）。

## 验收要点

1. 仅顶层任务可写 `schedule_rhythm`
2. `queued_auto_run` 入队；手动出队
3. 窗外 `deferred` + 文案
4. 出队顺序 depth ASC + FIFO
5. 排队额度与立即 auto_run/手动隔离（`queued_machine_slots` / `started_via`）
