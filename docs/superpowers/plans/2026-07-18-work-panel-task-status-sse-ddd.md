# DDD — Work Panel 任务状态 SSE

## 限界上下文

| 上下文 | 职责 |
|--------|------|
| Task（taskTaskService） | 任务聚合 SSOT；发布 TASK_STATUS_CHANGED |
| Realtime Delivery（taskSSE + taskEvents fanout） | 将状态变更投递到 workspace 订阅者 |
| Work Panel UI | 初载查询 + 增量投影 |

## 领域事件

- **已有**：`TASK_STATUS_CHANGED`（payload 含 workspace_id、progress_column_id、completed…）
- **副作用**：Redis 帧（非新 Kafka 事件类型）`event_name=task_status_changed`

## 端口-适配器

- Port：WorkspaceSsePublisher（Redis PUBLISH）
- Adapter：taskEvents `2_fanout_work_panel_sse`
- Read adapter：taskSSE EventSource hub
