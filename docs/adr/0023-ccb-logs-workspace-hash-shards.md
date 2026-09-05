# ADR-0023: 评论容器启动日志按 workspace_id 哈希分表

- **Status:** accepted
- **Date:** 2026-08-20
- **Author:** Trae AI
- **Deciders:** 工程团队（/goal 自动采用）

---

## Context

任务详情评论执行区展示的「启动日志」（排队、启动实例、调用云 API 等）持久化在 `cloud_comment_container_binding_logs`。该表是时间累积型，行数随工作空间与评论启动次数增长，且当前无 `workspace_id`、主键为 `AUTO_INCREMENT`。

产品要求：**表名根据工作空间 ID 分片**，使同一工作空间的日志落在同一物理表，避免全租户扫单表，并为后续水平扩展预留固定片数。

冷热分离元规则：禁止分片表使用 `AUTO_INCREMENT`；推荐预分 16/64/256 片。NFR 路径分片键：任务详情 URL 与 compute API 已携带 `workspaceId`，该键与访问模式对齐。

## Decision

We will physically shard comment-container-binding startup logs into **16 pre-created tables** named:

```text
cloud_comment_container_binding_logs_{00..15}
```

Shard index:

```text
CRC32(utf8(workspace_id)) % 16
```

Go uses `hash/crc32.ChecksumIEEE`; MySQL backfill uses `CRC32(workspace_id) % 16`. Table identifiers are **only** produced by `%02d` formatting — never by interpolating workspace strings into SQL.

Primary key is Snowflake `id VARCHAR(64)` (not AUTO_INCREMENT). Every row stores `workspace_id`. All read/write APIs and SSE persist paths must resolve a non-empty workspace ID before touching a shard.

Legacy table `cloud_comment_container_binding_logs` is retained as a backfill source and is no longer written by the application.

## Alternatives Considered

### Alternative 1: One table per workspace (`..._ws_{workspaceId}`)

- **Pros:** Table name literally contains workspace ID; no hash collision
- **Cons:** Unbounded DDL; MySQL table-count and metadata pressure; identifier sanitization risk
- **Why rejected:** Operationally unsafe; violates pre-split shard guidance

### Alternative 2: Keep one table, add `workspace_id` + RANGE partition by time

- **Pros:** Simpler ops; already used by `cloud_server_events`
- **Cons:** Does not change **table names**; single-table metadata/lock still shared
- **Why rejected:** Explicit requirement is table-name sharding by workspace

### Alternative 3: Hash-shard by `task_id` or `tenant_id`

- **Pros:** task_id always present on current inserts
- **Cons:** Same workspace’s logs scatter (task) or hotspot (large tenant)
- **Why rejected:** UI and URL are workspace-scoped; user specified workspace ID

## Consequences

### Positive

- Query/insert for a workspace hits one small table
- Fixed 16 tables, no runtime CREATE TABLE
- SQL backfill can use the same CRC32 as Go
- Frontend contract unchanged (`bindings[].logs`)

### Negative / Trade-offs

- Cross-workspace analytics must fan-out 16 tables (not a product path today)
- Empty `workspace_id` cannot be routed; must fail closed
- Legacy AUTO_INCREMENT ids are replaced by Snowflake on new rows; backfilled rows keep original numeric id as string
- `cloud_job_execution_event` still time-partitioned only (follow-up)

### Mitigations

- List/create already receive workspace from `handleCloudTaskRoutes` / workspace URL
- SSE persist resolves workspace from payload then CSC
- Identifier helper unit-tested for empty input and format `^cloud_comment_container_binding_logs_[0-9]{2}$`

## References

- 设计：`docs/superpowers/specs/2026-08-20-ccb-logs-workspace-shard-design.md`
- 冷热分离与分库分表元规则
- NFR 路径分片键：`.ai/01_project_constraints/48_nfr_path_shard_id_scalability.md`
