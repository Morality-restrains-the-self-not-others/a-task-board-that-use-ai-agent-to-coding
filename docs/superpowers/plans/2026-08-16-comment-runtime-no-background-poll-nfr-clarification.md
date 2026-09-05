# NFR 澄清 — 评论运行态推送同步

- **日期**: 2026-08-16
- **增量**: Push live + 按钮对账 + 删除 UI 轮询
- **默认等级**: L2 Standard（auto-flow）

## 路径分片键审视

| 路径 | 分片 ID | 适配性 | 动作 |
|------|---------|--------|------|
| GET `/api/cloud/compute/server-runtime-status/.../comment_id/{cid}` | tenant_id + workspace_id + task_id + comment_id | tenant_id 为租户分片键；comment_id 为评论实例路由键，基数高、访问对齐 | 保持；禁止无 comment_id |
| SSE `server_status_update`（按 task_id 订阅） | task_id 路由；payload.comment_id | task_id 适合任务内扇出；评论隔离靠 payload | 成功/停止必须带 comment_id |
| Kafka `CLOUD_SERVER_STOPPED` key=task_id | task_id | 与现有消费者一致；comment_id 在 envelope | 发布点写入 comment_id |
| 前端路由任务详情 | tenant + workspace + task | 合适 | 无新路由 |

可伸缩性：**L2**。直播改为推送后 Describe QPS 与开页用户数解耦；升级触发：单租户同时开页 > 500 且 SSE 连接成为瓶颈时再评估 taskSSE 分区。

## 类别定级

| 类别 | 等级 | 说明 |
|------|------|------|
| 性能 | L3 | 禁止 UI ticker；多评论展开不得并发 Describe |
| 可用性 | L2 | SSE 断连提示重连 + 按钮对账，不回退轮询 |
| 安全 | L2 | Describe 必须 comment 范围；无新公网接口 |
| 一致性 | L2 | 直播最终一致（推送）；按钮强对账云真源 |
| 可观测性 | L2 | SSE/HTTP 带 trace_id；前端 data-traceId |

## 质量场景

- **刺激**: 两条评论面板同时 Initializing 60s。**响应**: 0 次周期性 runtime-status。
- **刺激**: 启动成功 SSE 含 comment_id + Running。**响应**: 仅该面板更新，0 次自动 Describe。
- **刺激**: 用户点刷新。**响应**: 1 次 GET，path 含该 comment_id。
