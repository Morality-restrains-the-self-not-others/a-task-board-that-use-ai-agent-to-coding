# Launch checklist — queue schedule history

## Feature flag

无。增量挂在既有 `/queue-schedule/` 页；新 GET history 只读。未跑迁移前 history 表不存在，GET 快照 `recent_history` 为空数组（list 失败被吞）。

## Rollback

1. 前端不渲染 `ScheduleHistoryCard`（回退 taskFE）。
2. 停写 append（回退 TTS）；表可保留。

## 部署

1. `dataMigrate/taskTaskService/017_queued_schedule_history.sql` 经 9999 / migrate.sh
2. 精准编译重启 `task-task-service`、`taskFE`
3. 打开 `/tenant/{tid}/queue-schedule/?workspace_id=` 确认状态栏下有「调度历史」
