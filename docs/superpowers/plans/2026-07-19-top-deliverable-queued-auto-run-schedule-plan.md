# 实施计划：顶层交付物排队自动执行调度节奏

## 事件契约（先）

| 事件 | Topic（默认） | Payload 关键 |
|------|---------------|-------------|
| ScheduleRhythmUpdated | schedule-rhythm-updated | task_id, tenant_id, workspace_id, enabled, window |
| TaskQueuedForAutoRun | task-queued-for-auto-run | task_id, top_task_id, depth |
| TaskDequeuedFromAutoRun | task-dequeued-from-auto-run | task_id, reason |
| QueuedAutoRunDeferred | queued-auto-run-deferred | task_id, reason |
| QueuedAutoRunStarted | queued-auto-run-started | task_id, top_task_id |

## 任务清单

- [ ] TTS：DB 迁移 rhythm/queued 列 + `queued_auto_run_memberships`
- [ ] TTS：`queued_schedule.go` 窗口/深度/入队出队
- [ ] TTS：dispatcher ticker + 代调 start-vm-auto + started_via
- [ ] TTS：PATCH/GET 接线 + 事件 publish
- [ ] TTS：`queued_schedule_test.go`
- [ ] Cloud：`started_via` 列 + 写入 + 占用查询 API（或内部查询）
- [ ] Vue：顶层节奏表单 + 排队开关 + deferred 文案
- [ ] intents + table_ownership + value-stream（已部分完成）
- [ ] Swagger / 网关通配确认

## DDD 落点

无独立 `domain/` 包时：逻辑放 `taskTaskService/src/queued_schedule*.go`，概念见设计文档 §4。
