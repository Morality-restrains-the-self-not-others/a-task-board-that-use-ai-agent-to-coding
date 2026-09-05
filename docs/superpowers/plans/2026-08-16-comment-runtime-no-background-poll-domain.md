# DDD — 评论运行态推送同步

- **日期**: 2026-08-16
- **范围**: 不新建限界上下文；在 Cloud / Task UI 内明确直播 vs 对账端口。

## Bounded Contexts

- **Cloud Runtime**（taskCloudService / taskEvents cloudserverstopped）— 实例生命周期真源
- **Realtime Delivery**（taskSSE）— 按 task_id 投递，payload 带 comment_id
- **Task Detail UI**（taskFE）— 每评论 snapshot 展示

## Aggregates

- **CommentCloudServerConfig**（已有）— 根：comment_id 绑定的 CSC
- **CommentRuntimeSnapshot**（前端）— 仅 apply 推送或按钮回包

## Domain Events

| 意图 | 事件 | 发布点 | 消费者 | 例外 |
|------|------|--------|--------|------|
| 停机请求接受 | CLOUD_SERVER_STOPPED | compute_stop_vm | cloudserverstopped → SSE | — |
| 启动成功 | 既有 start SSE / STARTED | start_vm | taskFE apply | 不新增 MQ |
| 容器心跳 | container_heartbeat | 容器 | taskFE apply | 不新增 MQ |
| 用户刷新 | — | 按钮 GET | — | 纯查询 |

## Ports

- `LiveRuntimePush`：SSE status_data 必须可含 `comment_id` + `runtime_status`
- `ReconcileRuntimeQuery`：一次性 Describe，禁止 ticker 适配器
