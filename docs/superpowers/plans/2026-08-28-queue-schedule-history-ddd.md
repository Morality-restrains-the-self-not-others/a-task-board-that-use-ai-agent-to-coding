# 自动调度安排 · 调度历史 — 领域建模

- 日期：2026-08-28

## 限界上下文

**Task / Workspace Queue Schedule**（既有 taskTaskService）。不新建 BC。

## 聚合

- **WorkspaceScheduleRhythm**（既有）：增加 `last_in_window` 以检测窗口翻转。
- **QueuedAutoRunMembership**（既有）：入队/出队/启服仍由其命令发出。
- **QueuedScheduleHistoryEntry**（新，非聚合根）：append-only 记录；一致性边界是「主命令成功后再 best-effort 写入」。不以历史表回滚主命令。

## 值对象

- `ScheduleHistoryEventType`：rhythm_saved | member_enqueued | member_dequeued | member_started | window_entered | window_exited | auto_close_warned | auto_close_released
- `HistoryCursor`：`(created_at, id)`

## 领域事件

| 业务意图 | 事件 | 发布点 | 新/既有 |
|----------|------|--------|---------|
| 保存节奏 | WorkspaceScheduleSaved | PUT | 既有；副作用 append |
| 入队 | TaskQueuedForAutoRun | enqueue | 既有 |
| 出队 | TaskDequeuedFromAutoRun | dequeue | 既有 |
| 启服 | QueuedAutoRunStarted | startQueuedMembership | 既有 |
| 进入窗口 | WorkspaceScheduleWindowEntered | dispatch 翻转 | **新** |
| 离开窗口 | WorkspaceScheduleWindowExited | dispatch 翻转 | **新** |
| auto-close | ScheduleAutoCloseWarned/Released | 既有 | 既有 |
| 查询 | — | GET | 纯查询例外 |

## 不变式

1. 历史行的 workspace_id 必须等于命令所属工作空间。
2. 同一 workspace 连续两次 dispatch 若 in_window 未变，不得再写 window_*。
3. 查询不得返回其他 workspace 的行。
4. append 失败不得使 enqueue/dispatch 失败。
