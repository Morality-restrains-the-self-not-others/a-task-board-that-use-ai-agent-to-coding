# 评论启动日志 workspace 分表 — DDD

- **Date:** 2026-08-20
- **Service:** taskCloudService（既有 `package main` / `src/`，不新建独立 domain 包以免与现服务风格冲突）

## Bounded context

Cloud compute — CommentContainerBinding 启动时间线。

## 模型

- **Entity:** `CommentContainerBindingLog` — id (Snowflake), workspace_id, company_id, task_id, comment_id, binding_id, stage, message, created_at
- **Value object:** `LogShard` — index 0–15；`TableName` = `cloud_comment_container_binding_logs_%02d`
- **Domain service:** `ResolveLogShard(workspaceID) (LogShard, error)` — 空 ID 为领域错误
- **Repository port (逻辑):** `Append(log)` / `ListByTask(workspaceID, companyID, taskID)` — 实现选表后参数化 SQL

## 领域事件

无新事件。append-log 是既有 SSE 路径的持久化侧面。意图文档标注「无对应新事件」。

## 聚合

不把全部日志收进 Binding 聚合（行数无界）。Binding 仍为调度状态机聚合；Log 为附属时间序列，按 workspace 分片存储。
