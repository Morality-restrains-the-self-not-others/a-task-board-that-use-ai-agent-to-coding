# NFR 澄清 — ztree 执行日志服务端持久化

- **日期:** 2026-08-20
- **价值流:** `docs/superpowers/plans/2026-08-20-ztree-exec-log-server-persist-value-stream.md`
- **默认等级:** L2；非资金/配额/云资源创建，写路径不强制 L3

## 路径分片键审视

| 路径 | 已携带 ID | 分片键判定 | 动作 |
|------|-----------|------------|------|
| POST `.../tenant_id/{t}/workspace_id/{ws}/task_id/{task}/...` layer-graph-push | tenant, workspace, task；comment 在 CSC/body | **workspace_id 适合租户级分库**；task_id 对齐本页访问；comment 为实体后缀 | 落库 UNIQUE `(workspace_id, task_id, comment_id)`；一期 HASH 预留不分片 |
| GET `.../container-layer-graph/.../workspace_id/{ws}/task_id/{task}/comment_id/{cmt}/` | 同上 | 合适；禁止只带 job_id | 缺 ws/task/comment → 400 |
| GET `.../container-job-execution-log/` | 同上 + query job_id | 合适（023 已按时间分区；查询须带 task） | 本期不改 |
| GET `.../container-clone-log/` | 路径含 ws/task | 合适；权威在容器内存 | **不落库** |
| Kafka SSE_MESSAGE | 多按 task_id | 保持；payload 带 workspace_id | 快照事件 key = comment 聚合键 |
| LayerGraphSnapshotPersisted | workspace+task+comment | 合适 | 无自动消费者 |
| FE 任务详情路由 | workspace/task 在页上下文 | 合适 | 无 |

无「缺分片键却声称无可伸缩性」的新公网路径。快照表年增量约「评论数」级，L0 分片（HASH 预留）。可伸缩性类别仍纳入（路径已带键且判定合适）。

## 幂等性审视

| 路径 | 副作用 | 重复触发源 | 业务重复边界 | 幂等键 | 重放语义 | 等级 |
|------|--------|------------|--------------|--------|----------|------|
| layer-graph-push | UPSERT 快照 + SSE + 领域事件 | 容器重试、重复 PUSH | 同一评论最新树 | `(workspace_id, task_id, comment_id)` UNIQUE UPSERT | 后写覆盖；行数不增 | L2 |
| GET layer-graph / job-log | 无 | — | — | — | 纯查询 | L0 |
| GET clone-log | 无（不写库） | — | — | — | 代理容器 | L0 |
| LayerGraphSnapshotPersisted 消费 | 本期无消费者 | Kafka at-least-once | 同评论快照 | 若二期消费须同三键 | 禁止用 tenant/user 当键 | n/a |
| job-stream-push | 既有 023 | 既有 | `(task_id, job_id, seq)` | 保持 | L2 既有 |

禁止用单独 `tenant_id` / `user_id` 作快照幂等键。非点击按钮写路径，无新 FE `Idempotency-Key`（PUSH 来自容器）。

## 类别定级

| 类别 | 级别 | 说明 |
|------|------|------|
| 可伸缩性 | L1 | 小表 last-write-wins；workspace 预留 HASH |
| 数据一致性 | L2 | 同评论覆盖；SSE 与库允许短暂并发（直播仍 SSE） |
| 容错 | L2 | 落库失败打 ERROR，仍发 SSE，避免丢直播 |
| 安全 | L2 | 三键 IDOR；日志不打整棵 graph |
| 可用性 | L2 | 关容器 GET 200；无行空图 |
| 性能 | L2 | 单行 JSON，远小于 1MB |
| 可观测性 | L2 | UPSERT/GET/失败 structured + trace_id |

## 质量场景

1. 刺激：合法 PUSH 后停容器再 GET。响应：200，layers 与最后一次 PUSH 一致。
2. 刺激：同键两次 PUSH。响应：一行，graph 为第二次。
3. 刺激：GET 缺 task。响应：400。
4. 刺激：路径 workspace 与落库不同。响应：空图。
5. 刺激：FE endpoint=false。响应：仍请求 layer-graph。

## 领域模型影响

聚合根 `LayerGraphSnapshot`，身份 `(workspace_id, task_id, comment_id)`。值对象：`graph_json`（layers/jobs 数组文档）。无跨聚合事务。事件在 UPSERT 成功后发出；失败不发领域事件（SSE 仍可发）。
