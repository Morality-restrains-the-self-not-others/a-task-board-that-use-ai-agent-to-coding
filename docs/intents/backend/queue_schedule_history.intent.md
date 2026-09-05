# 意图：工作空间排队调度历史持久化与查询

- 日期：2026-08-28
- 设计：`docs/superpowers/specs/2026-08-28-queue-schedule-history-design.md`

## 范围

- 表 `task_queued_schedule_history`（月分区）
- 状态变化时 fail-open append
- GET `/api/tenant/{tid}/workspace/{wid}/queue-schedule/history/`
- GET 快照增加 `recent_history`（≤8）
- 窗口翻转事件 `WorkspaceScheduleWindowEntered` / `WorkspaceScheduleWindowExited`

## 业务意图 → 事件对照

| 业务意图 | 事件名（过去式） | 发布点 | 消费者/副作用 | 例外 |
|---------|----------------|--------|--------------|------|
| 保存节奏 | WorkspaceScheduleSaved | `applyWorkspaceScheduleRhythmFromBody` | append `rhythm_saved` | 既有事件 |
| 入队 | TaskQueuedForAutoRun | `enqueueQueuedAutoRun` | append `member_enqueued` | 既有 |
| 出队 | TaskDequeuedFromAutoRun | `dequeueQueuedAutoRun` | append `member_dequeued` | 既有 |
| 调度启服 | QueuedAutoRunStarted | `startQueuedMembership` | append `member_started` | 既有 |
| 进入允许时段 | WorkspaceScheduleWindowEntered | `dispatchWorkspaceQueueWith` 翻转 | append `window_entered` | 新事件 |
| 离开允许时段 | WorkspaceScheduleWindowExited | 同上 | append `window_exited` | 新事件 |
| auto-close 预告 | ScheduleAutoCloseWarned | 既有 | append `auto_close_warned` | 既有 |
| auto-close 释放 | ScheduleAutoCloseReleased | 既有 | append `auto_close_released` | 既有 |
| 查询历史 | — | GET | — | 纯查询 |

## 验收

1. 无 workspace 访问 → 403
2. 列表按 created_at DESC 分页，不跨 workspace
3. 空扫不写行；翻转只写一行 window_*
4. append 失败不阻断入队/分发
