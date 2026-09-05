# DDD — LayerGraphSnapshot（ztree 层图快照）

- **Date:** 2026-08-20
- **NFR:** `docs/superpowers/plans/2026-08-20-ztree-exec-log-server-persist-nfr-clarification.md`

## Bounded Context

**Cloud Resource / Comment execution**（`taskCloudService`）。容器 runtime 是临时执行者，不是日志 owner。

沿用 Cloud 现有 **main 包 store + `domain/` 纯函数** 模式（与 `running_counts`、023 job event 同构）。不新建独立 hex 包，避免与 job_execution_event 分裂。

## Aggregate

**LayerGraphSnapshot**

| 字段 | 规则 |
|------|------|
| Identity | `workspace_id` + `task_id` + `comment_id` 均非空 |
| `graph_json` | JSON 对象，须含 `layers`、`jobs` 数组 |
| last-write-wins | 同身份覆盖 |

不变量：禁止无 workspace 或 task 的快照。空树合法（`layers:[]`）。

## Domain events

`LayerGraphSnapshotPersisted` — UPSERT 成功后由应用服务 `publishDomainEvent`。无本期消费者。Topic：`layer-graph-snapshot-persisted`。

## Ports（逻辑，实现落 src store）

- `Upsert(identity, graphJSON)`
- `Get(identity) -> (doc, found)`

基础设施：MySQL JSON 列；Snowflake `id`；utf8mb4。

## 非本聚合

- Job 步骤：既有 `JobExecution` / `cloud_job_execution_event`
- 克隆日志：无服务端实体

## 纯领域代码

`taskCloudService/domain/layer_graph_snapshot.go`：身份校验 + 空图文档，不 import database/sql。
