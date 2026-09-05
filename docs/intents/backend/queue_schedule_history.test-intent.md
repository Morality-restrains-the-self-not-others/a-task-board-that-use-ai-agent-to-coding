# 测试意图：工作空间排队调度历史

对应：`docs/intents/backend/queue_schedule_history.intent.md`

## 覆盖

`taskTaskService/src/queued_schedule_history_test.go`、`queued_schedule_workspace_test.go`、dispatch/membership 回归。

## 用例

| ID | 给定 | 当 | 则 |
|----|------|----|----|
| T1 | 入队 | enqueue | 一行 `member_enqueued` |
| T2 | PUT 节奏 | save | 一行 `rhythm_saved` |
| T3 | last_in_window=1 且现窗外 | dispatch | 一行 `window_exited`；无 N 条 deferred 历史 |
| T4 | 同窗口再 dispatch | — | 不再写 window_* |
| T5 | GET history | 有权限 | 本 workspace 新→旧 |
| T6 | GET history | 无 workspace 权限 | 403 |
| T7 | GET 快照 | 有行 | `recent_history` ≤8 |
| T8 | cursor | 第二页 | 不重复第一页 |
