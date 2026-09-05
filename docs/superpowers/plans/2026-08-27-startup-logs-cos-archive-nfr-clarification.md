# NFR 澄清：评论启动日志 COS 归档

- **Date:** 2026-08-27
- **Value stream:** `docs/superpowers/plans/2026-08-27-startup-logs-cos-archive-value-stream.md`

## 路径分片键强制审视

| 路径 | 分片 ID | 适配性 | 动作 |
|------|---------|--------|------|
| 工作台 `/tenant/{tenantId}/work-panel` | tenantId | 租户隔离，日志按 workspace 分片（ADR-0023） | 无补键；COS key 含 workspaceId |
| CCB list `.../workspace/{workspaceId}/task/{taskId}/comment-container-bindings` | workspaceId | 与 shard CRC32(workspace_id) 对齐 | 查询必须带 workspace_id |
| COS object key | workspaceId+taskId+commentId | 评论级对象，基数足够 | 禁止 bucket 根列举 |
| CommentStartupLogArchived Kafka key | log_id（含 ws/task/comment 前缀） | 与业务重复边界（单条日志）同粒度 | publish-only |
| PATCH step-full-cos | 无租户 ID（平台配置） | L0 平台单例 | 升级触发：多区域 COS 时拆 conf |

## 幂等性强制审视

| 路径 | 副作用 | 重复触发源 | 业务重复边界 | 幂等键 | 重放语义 |
|------|--------|------------|--------------|--------|----------|
| insertCCBLogRow | MySQL insert | SSE/调度重试 | 一行日志 | Snowflake log id（新行） | 新 id 即新行 |
| persist COS merge | PutObject | 同 id 重放 / 并发 append | 评论 bundle 内一条 log | log.id | 已存在则跳过/覆盖同 id |
| 指针 UPSERT | 一行/评论 | 并发写 | 评论 | UNIQUE(workspace,task,comment) | 覆盖 object_key/etag |
| CommentStartupLogArchived | Kafka | 同 log 再归档 | 该 log 行 | ws:task:comment:log_id | 无消费者；未来 Ack 空操作 |
| PATCH COS conf | 写 yaml | 双击保存 | 平台配置文档 | 前端 Idempotency-Key（既有） | 后写覆盖 |
| list hydrate | 无副作用 | — | — | L0 | 只读 |

资金/云资源：本增量不扣费、不调云启停。L2。

## 类别支撑程度（默认 L2）

| 类别 | 级别 | 说明 |
|------|------|------|
| 可伸缩性 | L2 | 热表 16 片；COS 每评论一对象；指针表非时间累积 |
| 数据一致性 | L2 | MySQL 先写；Put 成功后驱逐分片；读路径 COS 优先 |
| 容错 | L2 | COS 失败不阻断热写；读回退分片 |
| 安全 | L2 | staff 配密钥；SSE-COS；直连无 Proxy |
| 可观测性 | L2 | `ccb_startup_log_archive_ok/err` + object_key/bytes |

## 质量场景

1. **刺激**：启动过程写 20 条日志后 COS 短暂 503。**响应**：面板仍显示 20 条（分片）；恢复后后续行归档；日志含 archive_err。
2. **刺激**：Put 成功后冷打开。**响应**：分片无该 id；list 从指针+COS 还原，条数与归档一致。
3. **刺激**：同 log id persist 两次。**响应**：bundle 仍 1 条。
