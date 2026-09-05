# ADR-0045: 评论启动日志双写腾讯云 COS

- **Status:** accepted
- **Date:** 2026-08-27
- **Author:** cursor
- **Deciders:** /goal 自动采用（页面元素：启动日志）

---

## Context

工作台「启动日志」持久化在 ADR-0023 的 workspace 哈希分片表。分片适合热 append，但对象体积会随启动次数增长，且分区裁剪后冷打开会丢时间线。ztree 执行全文已按 ADR-0039 归档 COS。产品要求启动日志**也**进入 COS。

## Decision

We will dual-write each comment's startup log timeline to Tencent COS as one JSON bundle per comment, reusing the step_full COS client (same bucket/credentials, distinct `startupLogsPathRule`). MySQL shards are a **write-ahead buffer**: after a successful object-store Put, the corresponding shard row is **deleted**. List APIs read the COS bundle first (via pointer table `cloud_comment_startup_log_object`) and only union leftover shard rows (Put failed / in-flight). COS Put is best-effort after a successful shard insert; Put failure must not delete the shard row.

Default key:

```text
workspace_{workspaceId}/task_{taskId}/comment_{commentId}/startup_logs.json
```

Merge is idempotent on log Snowflake `id`. SSE-COS AES256. HTTP client must not use env proxy.

## Alternatives Considered

### Alternative 1: COS as sole authority (no MySQL buffer)

- **Pros:** One store.
- **Cons:** Every log line is a COS RMW on the hot path with no local durability if Put fails; contradicts ADR-0023 append isolation.
- **Why rejected:** 启动过程仍需要低延迟 append；Put 失败时必须能从分片读出。采用「分片缓冲 → Put 成功后驱逐」保留热写与失败回退。

### Alternative 2: Archive only on terminal binding status

- **Pros:** Fewer Puts.
- **Cons:** Crash/refresh mid-start loses lines.
- **Why rejected:** 用户要求启动日志也在 COS，覆盖全时间线。

### Alternative 3: One COS object per log line

- **Pros:** 无 merge。
- **Cons:** 对象数爆炸。
- **Why rejected:** 运维与 list 成本不可接受。

## Consequences

### Positive

- 冷打开从 COS 还原；热表不再长期堆积启动日志。
- 与 step_full 共用密钥与直连客户端。

### Negative / Trade-offs

- COS 滞后于 MySQL（最终一致）。
- 并发 append 需进程内 mutex 串行 RMW。

### Mitigations

- 读路径 COS 优先、分片仅补缺口；COS 失败不阻断热写、不删分片。
- 指针 UNIQUE(workspace,task,comment) 便于 task 级 hydrate。

## References

- [ADR-0023](0023-ccb-logs-workspace-hash-shards.md)
- [ADR-0039](0039-ztree-step-full-cos-archive.md)
- [ADR-0006](0006-tencent-cos-vendor-documents.md)
- `docs/superpowers/specs/2026-08-27-startup-logs-cos-archive-design.md`
