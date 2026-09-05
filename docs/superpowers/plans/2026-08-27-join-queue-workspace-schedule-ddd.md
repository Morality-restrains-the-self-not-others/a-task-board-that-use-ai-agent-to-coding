# DDD：入队前确认工作空间自动调度

无新聚合。复用：

- **工作空间调度节奏**（`workspace_schedule_rhythms.enabled`）— 是否「启动了自动调度」。
- **排队成员**（`task_queued_auto_run_memberships`）— 仅在 enabled 时由本 UI 写入。

前端应用服务步骤：`CheckWorkspaceAutoScheduleEnabled`（读端口 GET queue-schedule）→ `EnqueueTaskForAutoRun`（既有 PATCH）。

无新领域事件。纯查询例外已写入意图对照表。
