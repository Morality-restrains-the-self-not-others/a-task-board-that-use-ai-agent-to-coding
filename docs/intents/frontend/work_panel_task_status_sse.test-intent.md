# 测试意图：工作面板任务状态 SSE

对应：`work_panel_task_status_sse.intent.md`

| ID | 场景 | 期望 | 类型 |
|----|------|------|------|
| T1 | normalizeInboundMessage 含 workspace hub | hub key = `workspace:{ws}` | taskSSE 单测 |
| T2 | GET work-panel-events-sse 无 gateway secret | 403 | taskSSE 单测 |
| T3 | GET work-panel-events-sse 有 secret | 200 + connected 帧 | taskSSE 单测 |
| T4 | taskEvents fanout 收到 TASK_STATUS_CHANGED | Redis publish `sse:workspace:{ws}` | Go 单测 |
| T5 | 前端收到 task_status_changed | todos 中卡片 progress_column_id 更新 | Vitest |
| T6 | 前端收到 heartbeat | todos 不变 | Vitest |
| T7 | 前端收到 task_created / task_deleted | 调用 onNeedResync（fetchTodos） | Vitest |
| T8 | taskTaskService 创建任务 | 发布 TASK_CREATED | Go 单测 |
| T9 | taskTaskService 删除任务 | 发布 TASK_DELETED | Go 单测 |
