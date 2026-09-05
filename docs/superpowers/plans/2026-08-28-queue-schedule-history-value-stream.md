# 自动调度安排 · 调度历史 — 价值流

- 日期：2026-08-28

## Related Value Streams

- `top-deliverable-queued-auto-run-schedule`：入队/出队/分发。本增量把这些事实投影为可读历史。
- `workspace-queue-schedule`：节奏页与状态栏。本增量在状态栏下补时间线。
- `workspace-queue-schedule-members-manage`：排队卡管理。本增量不改该卡。

## 用户价值

在自动调度安排页看到「调度做过什么」，而不只看到此刻是否在时段内。

## 增量（单一 MVP）

1. 状态变化写入 `task_queued_schedule_history`
2. 快照带 `recent_history`；历史卡展示
3. 加载更多分页 GET

无第二期（不过滤类型、不导出）。

## 步骤与测试点

| 步骤 | 测试文件 | 测试点 |
|------|----------|--------|
| 打开页看状态栏+历史 | WorkspaceQueueSchedule.test.js | 状态栏与 history-card 同时存在 |
| 有 recent_history | ScheduleHistoryCard.test.js | 行文案/时间 |
| 空历史 | ScheduleHistoryCard.test.js | empty 文案 |
| 加载更多 | useWorkspaceQueueSchedule.test.js | GET history + cursor |
| 入队落历史 | queued_schedule_history_test.go | member_enqueued |
| 窗口翻转 | queued_schedule_history_test.go | 一行 window_exited |

## YAML

`conf/value-stream.yaml` 增加 `workspace-queue-schedule-history`。
