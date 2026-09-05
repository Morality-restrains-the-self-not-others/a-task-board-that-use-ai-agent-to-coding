# ADR-0007: 任务级 CSC 只标记运行中机器数与容器数

- **Status:** accepted
- **Date:** 2026-08-14
- **Author:** cursor
- **Deciders:** /1-brainstorming 总体设计审批（用户确认可清空旧数据、不兼容存量任务）

---

## Context

评论已各自拥有 CSC（`UNIQUE(workspace_id, task_id, comment_id)`），运行态 API 已按 `comment_id` 读取。任务级行（`comment_id=''`）仍被写成「这台任务的服务器」：`last_runtime_status` 按 `task_id` 整表刷、启动成功无 `comment_id` 时把 `instance_id` 写到模板行、看板用一份 Starting/Running 盖住所有评论。

用户要求：任务级不再存服务器生命周期，只标记该任务有多少台评论机器在运行、多少个评论容器在运行。存量任务级运行态可直接清空，禁止再从任务级领养 instance。

## Decision

We will keep the task-level CSC row as **hardware/platform template + two counters**, and treat comment CSC as the only runtime authority:

1. Columns `running_machine_count` / `running_container_count` on `cloud_server_configs` (template row only; comment rows stay 0).
2. Machine running = `comment_id!=''` and `machineRuntimeCountsAsStarted` (Running or `mock-`); Starting does **not** count. Container running = same plus non-empty `server_url`.
3. `persistStartVmInstanceBinding` without `csc_id`/`comment_id` is a no-op + warn. Import of template rows strips runtime fields.
4. `setCloudServerLastRuntimeStatus` requires `comment_id` or `instance_id`; never `WHERE task_id=?` alone.
5. Delete `healCommentCSCInstanceFromTaskLevel`. DDL 019 clears template `instance_id` / status / IP / URL and backfills the two counts.
6. No new HTTP path; extend `workspace-runtime-indicators` JSON. No Kafka event (`TaskRunningCountsRecomputed` is same-DB projection, exempt).

## Alternatives Considered

### Alternative 1: 只读时 COUNT，不落列

- **Pros:** 无 DDL。
- **Cons:** 列表/任务头没有稳定标记字段。
- **Why rejected:** 用户要求「标记」。

### Alternative 2: 任务级仍存 Starting/Running 再派生数量

- **Pros:** 改动面小。
- **Cons:** 正是串台根因。
- **Why rejected:** 与目标冲突。

### Alternative 3: 保留一轮 heal 再清空任务级 instance

- **Pros:** 存量评论可自动领养误写的任务级 instance。
- **Cons:** 延长双读窗口；用户已确认可清空旧数据。
- **Why rejected:** D6 明确删除 heal。

## Consequences

### Positive

- 多评论并行启动/停机不再互相覆盖运行态。
- 看板布尔与两计数同源（评论行扫描）。
- 任务级模板职责清晰，可继续给新评论克隆 platform/region/auth。

### Negative / Trade-offs

- 仅有任务级 instance、评论行仍空的存量任务显示 0/0，需重新启动。
- idle reuse 仍可能读到历史任务级 instance 行（本增量未改闲置复用数据源）。

### Mitigations

- DDL 019 同事务清空模板运行态并回填计数。
- 启动/状态/公网 IP 写路径后重算两列；日志 `event=task_running_counts_updated`。

## References

- [设计](../superpowers/specs/2026-08-14-task-level-running-comment-server-count-design.md)
- [DDD](../superpowers/specs/2026-08-14-task-level-running-comment-server-count-ddd.md)
- [计划](../superpowers/plans/2026-08-14-task-level-running-comment-server-count-plan.md)
- 意图：`docs/intents/backend/task_level_running_comment_server_count.intent.md`
