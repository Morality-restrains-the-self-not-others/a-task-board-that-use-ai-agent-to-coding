# DDD：任务与项目内容历史版本

- 日期：2026-08-30
- NFR：`docs/superpowers/plans/2026-08-30-task-project-entity-revisions-nfr-clarification.md`

本仓 taskTaskService / taskProjectService 以 `src/` 为主；领域规则以纯函数落在同服务 `domain/`（无基础设施 import）。

## Bounded Contexts

| Context | Owner 服务 | 说明 |
|---------|------------|------|
| Task Collaboration | taskTaskService | Task 聚合 + TaskRevision |
| Project Catalog | taskProjectService | Project 聚合 + ProjectRevision |

禁止跨库写对方 revision 表。

## Aggregates

```
Task (root: task_id)
  live: title, description, ...
  revisions: TaskRevision[]  // 仅追加

Project (root: project_id)
  live: name, description, ...
  revisions: ProjectRevision[]
```

一致性：单次创建/更新事务内最多追加 1 条 revision。

## Entities / VOs

- `TaskRevision`：id, tenant_id, workspace_id, task_id, version_num, title, description, actor_user_id, changed_fields, created_at — 创建后不可变
- `ProjectRevision`：对称，name 替代 title，无 workspace_id
- VO `ChangedFields`：有序集合 {title|description} 或 {name|description}

## Domain services（纯函数）

- `DiffVersionedTask(old, new) (changed []string, record bool)`
- `NextVersion(max int) int`
- `ShouldRecordCreate() bool` → 恒 true（写 v1）

## Ports

- `RevisionRepository.Insert(tx, rev) error`
- `RevisionRepository.ListByEntity(entityID, tenantID, limit, offset) (rows, total, error)`
- `RevisionRepository.Get(entityID, tenantID, revisionID) (*Rev, error)`
- `EventPublisher.PublishRevisionRecorded(ctx, evt) error`

适配器：MySQL（分区表）、Kafka（commit 后）。

## Domain events

```
TaskRevisionRecorded { tenant_id, workspace_id, task_id, revision_id, version_num, actor_user_id, changed_fields, trace_id }
ProjectRevisionRecorded { tenant_id, project_id, revision_id, version_num, actor_user_id, changed_fields, trace_id }
```

幂等键 = `revision_id`。消费者不写 owner 库。

## 目录（约定）

```
taskTaskService/domain/task_revision.go
taskTaskService/src/task_revision_store.go
taskTaskService/src/task_revision_handlers.go
taskProjectService/domain/project_revision.go
taskProjectService/src/project_revision_store.go
```
