# 测试意图：自动调度安排页展示调度历史

对应：`docs/intents/frontend/queue_schedule_history.intent.md`

## 覆盖

| 层 | 文件 |
|----|------|
| 页面 | `WorkspaceQueueSchedule.test.js` |
| 卡片 | `ScheduleHistoryCard.test.js` |
| composable | `useWorkspaceQueueSchedule.test.js` |

## 用例

| ID | 给定 | 当 | 则 |
|----|------|----|----|
| T1 | 快照含 recent_history | 打开页 | `schedule-history-card` 出现对应文案；状态栏仍在 |
| T2 | recent_history 空 | 打开页 | `schedule-history-empty` 可见 |
| T3 | has_more | 点加载更多 | GET `.../queue-schedule/history/` 带 cursor |
| T4 | history GET 失败且有 X-Trace-Id | — | 错误节点 `data-traceId` |
| T5 | 行含 task_id | — | 标题为真实 `a[href]` |
