# ADR-0008: 任务帖人读序号与技术主键分离

- **Status:** accepted
- **Date:** 2026-08-14
- **Author:** cursor
- **Deciders:** /1-brainstorming 总体设计审批（用户确认存量清空、展示 `#N`）

---

## Context

任务帖技术 ID 由 `genID("task")`（纳秒时间戳）生成，前端用后 6 位做人读编号。同一工作空间内后 6 位无序。项目强制主键用全局唯一 ID（Snowflake 规范），禁止工作空间自增当主键。`order_num` 是看板排序，不能当身份编号。

## Decision

We will keep the technical primary key (`task_tasks.id`) unchanged and add a **workspace-scoped, immutable display sequence**:

1. Column `workspace_seq INT UNSIGNED NOT NULL` on `task_tasks`, unique per `(tenant_id, workspace_id, workspace_seq)`.
2. Allocator table `task_workspace_seq` (`tenant_id`, `workspace_id`, `next_val`); allocate in the same transaction as task insert with `SELECT … FOR UPDATE`.
3. Human display is `#N` (variable width). APIs and Kafka keep using `id`; `TASK_CREATED` adds `workspace_seq`.
4. Deleted numbers are not reused. `order_num` remains kanban order only.
5. Existing task-post rows in `taskTaskService` are wiped by migration (not backfilled). `task_git_identities` and cross-service CSC are not wiped by this DDL.
6. No new HTTP path. Implementation lives in `taskTaskService` + `taskFE`.

## Alternatives Considered

### Alternative 1: 改 genID / Snowflake 让后几位递增

- **Pros:** 不改表。
- **Cons:** 时间戳低位无法表达工作空间序；改 PK 破坏外键与分片。
- **Why rejected:** 与全局唯一 ID 规范冲突，且不能按工作空间从 1 数。

### Alternative 2: 工作空间 AUTO_INCREMENT 当主键

- **Pros:** 发号简单。
- **Cons:** 分片冲突；违反 Snowflake 元规则。
- **Why rejected:** 硬约束。

### Alternative 3: 前端按 created_at 排名

- **Pros:** 无 DDL。
- **Cons:** 删除后编号漂移；无法稳定复制。
- **Why rejected:** 人读编号必须稳定。

### Alternative 4: Redis INCR 或无锁 MAX+1

- **Pros:** 实现短。
- **Cons:** 双写或并发撞号。
- **Why rejected:** 发号真源必须在任务库事务内。

## Consequences

### Positive

- 同一工作空间内人读编号有序、可口述、可搜索。
- 技术 ID 与跨服务引用不变。
- 发号与建帖同事务，一致性清晰。

### Negative / Trade-offs

- 存量任务帖被清空，需重建。
- 跨服务可能残留指向已删 `task_id` 的 CSC / 令牌。
- 删除造成序号空洞（与 GitHub 相同）。

### Mitigations

- 迁移清单明确：只清任务帖域，保留 git 身份。
- 跨服务孤儿行由 9999 / 运维处理，不在本 DDL 级联。
- 搜索强制工作空间作用域，避免序号撞车。

## References

- [设计](../superpowers/specs/2026-08-14-workspace-task-display-seq-design.md)
- 意图：`docs/intents/backend/workspace_task_display_seq.intent.md`
- 主键规范：`.ai/01_project_constraints/35_snowflake_id_generation.md`
