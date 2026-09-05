# DDD: 工作空间任务帖人读序号

> Design + NFR + value stream：2026-08-14-workspace-task-display-seq-*

## Bounded Context

Task（`taskTaskService`）。不新建上下文。

## Entities / VOs

| 概念 | 类型 | 说明 |
|------|------|------|
| `Task` | Entity / Aggregate Root | `id` 技术身份；`WorkspaceSeq` 人读身份 |
| `WorkspaceSeq` | Value Object | 正整数，工作空间内唯一，创建后不变 |
| `WorkspaceTaskSeq` | 发号设施 | `(tenant_id, workspace_id) → next_val`，非聚合根 |

## Invariants

1. 同一 `(tenant_id, workspace_id)` 内 `workspace_seq` 唯一。
2. 客户端不可指定 seq；只由发号器分配。
3. 删除不回收；回滚事务则不提交号。
4. 搜索按 seq 必须带 tenant + workspace（或调用方已限定的 workspace 列表）。

## Ports

- `WorkspaceSeqAllocator.Next(tenantID, workspaceID) (int, error)` — 由 MySQL 适配器在同一 `*sql.Tx` 实现。
- 既有 `EventBus.Publish(TASK_CREATED)` — payload 增补 `workspace_seq`。

## Domain Events

| 意图 | 事件 | 载荷增量 |
|------|------|----------|
| 创建任务帖 | `TASK_CREATED` | `workspace_seq` |

查询/展示：无事件。
