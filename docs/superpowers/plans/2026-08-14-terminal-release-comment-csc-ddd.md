# DDD：终态按评论 CSC 释放

- 日期：2026-08-14
- 增量：单增量（list + 按评论释放）

## 限界上下文

- **Cloud Runtime**（taskCloudService）：CSC 所有权
- **Task Events**（taskEvents）：终态编排

## 实体 / VO

- CommentCloudServerConfig（comment_id≠''）：运行实例
- TaskLevelTemplate（comment_id=''）：模板 + 计数，非实例
- TerminalKind：completed | cancelled
- Releaseable：machineRuntimeCountsAsStarted 或 server_url≠''

## 聚合

- 编排根：TaskId。一致性：逐评论释放，允许部分已停（幂等）。

## 端口（已有 + 新增）

- `TaskRuntimeLister.ListByTask(tenant, workspace, task) → comments + counts`
- `EventPublisher.PublishEvent`（复用）
- `LocalStopper` / `ContainerNotifier` / `InstanceMigrator`（复用）

领域层不新建独立 Go package：落在既有 `taskstatuschanged` + `cloudconfig` 端口，避免空骨架。

## 领域事件

| 意图 | 事件 | 发布点 | 消费者 |
|------|------|--------|--------|
| 任务终态 | TaskStatusChanged | taskTaskService（已有） | release-servers |
| 释放一台评论机 | CloudServerStopped | release-servers | cloudserverstopped |
| 无运行资源 | — | — | 例外：无状态变更 |

## 领域服务

`ReleaseCommentServersOnTerminal`：list → filter releaseable → graceful/stop → publish stopped。
