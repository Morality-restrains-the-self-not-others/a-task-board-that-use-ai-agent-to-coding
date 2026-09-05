# NFR — Work Panel 任务状态 SSE

| 类别 | 级别 | 说明 |
|------|------|------|
| 延迟 | L2 | 状态变更到他端可见：通常 < 3s（Kafka+Redis 路径） |
| 可用性 | L2 | SSE 断线不影响看板只读；可重连 + 可选 fetchTodos 对账 |
| 安全 | L3 | 网关 token + internal secret；无敏感字段 |
| 可观测 | L2 | 连接/扇出/投递日志带 traceId；level 小写 |
| 容量 | L2 | 计入 taskSSE maxConnections；workspace 级共享连接 |
| 一致性 | L2 | 最终一致；漏事件靠重连后 fetchTodos |

质量场景：双标签拖拽同步；网关 secret 缺失 403；超连接上限 503。
