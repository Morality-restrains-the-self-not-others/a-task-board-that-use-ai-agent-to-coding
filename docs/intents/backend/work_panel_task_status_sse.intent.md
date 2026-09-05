# 后端：Work Panel 任务状态 SSE 扇出

## 意图

当任务进度列/完成态变更，或任务被创建/删除时，须将变更扇出到同 workspace 的看板 SSE 订阅者（覆盖 Chrome 插件等跨端创建场景）。

## 行为

1. `taskTaskService` 在 progress/completed 变更时发布 `TASK_STATUS_CHANGED`；在创建成功后发布 `TASK_CREATED`；在删除成功后发布 `TASK_DELETED`。
2. intent `task_status_changed/2_fanout_work_panel_sse` 订阅上述三类事件，向 Redis `sse:workspace:{workspace_id}` 发布帧（`task_status_changed` / `task_created` / `task_deleted`）。
3. `taskSSE` 提供 `GET .../work-panel-events-sse/`，hub key `workspace:{workspace_id}`。
4. 禁止在 taskTaskService HTTP handler 内同步写 Redis/SSE。

## 业务意图 → 事件对照

| 业务意图 | 事件名（过去式） | MQ类型/契约 | 发布点 | 消费者/副作用 | 例外理由 |
|---------|----------------|------------|--------|--------------|---------|
| 任务进度列/完成态变更 | TaskStatusChanged | TASK_STATUS_CHANGED | taskTaskService（既有 publish） | 1_release_servers_on_terminal；2_fanout_work_panel_sse → Redis → taskSSE 看板 | — |
| 任务创建（含 Chrome 插件） | TaskCreated | TASK_CREATED | taskTaskService create | 2_fanout_work_panel_sse → Redis → taskSSE；前端 resync todos | — |
| 任务删除 | TaskDeleted | TASK_DELETED | taskTaskService delete | 2_fanout_work_panel_sse → Redis → taskSSE；前端 resync todos | — |
| Work Panel SSE 订阅（只读长连接） | — | — | — | — | 无对应事件：纯订阅/查询通道，不产生业务状态变更 |
