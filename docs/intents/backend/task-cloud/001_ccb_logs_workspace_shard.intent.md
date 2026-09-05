# 评论容器启动日志按工作空间分表

## 背景与目标

任务详情启动日志持久化表随工作空间膨胀。表名须由 workspace_id 分片，读写只打对应物理表。

## 范围与边界

- 范围内：`cloud_comment_container_binding_logs_*`、taskCloudService 读写
- 范围外：前端 DOM；`cloud_job_execution_event` / `cloud_server_events` 表名

## 约束与风险

- 表名禁止拼接用户字符串；CRC32%16；Snowflake PK
- 空 workspace 不得写入

## 验收标准

1. 16 张预创建表存在
2. 同 workspace 读写一致；跨 workspace 不可见
3. bindings list 的 logs 仍可冷打开还原

## 实施计划

见 `docs/superpowers/plans/2026-08-20-ccb-logs-workspace-shard-plan.md`

## 意图 → 事件

| 意图 | 事件名 | 发布点 | 消费者 | MQ类型/契约 |
|------|--------|--------|--------|-------------|
| 无新业务意图（存储路由） | 无对应事件 | — | — | 例外：纯存储选片；启动进度仍走既有 `publishTaskSSE`，不新增领域事件 |

变更记录：2026-08-20 新增。
