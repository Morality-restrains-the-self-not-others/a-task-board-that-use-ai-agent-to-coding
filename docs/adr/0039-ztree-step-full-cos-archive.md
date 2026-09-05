# ADR-0039: ztree 执行全文以腾讯云 COS 归档 step_full.json

- **Status:** accepted
- **Date:** 2026-08-23
- **Author:** cursor
- **Deciders:** /goal 自动采用（页面元素调整：层级落库 + COS 复查）

---

## Context

层图已落入 `cloud_layer_graph_snapshot`。Agent 步骤摘要在 `cloud_job_execution_event`。完整 `agent_step_full.json` 仅在容器磁盘，释放后无法复查执行日志。厂商证照已用腾讯云 COS（ADR-0006），执行日志需要独立路径规则与管理员可配参数。

## Decision

We will:

1. Keep ztree hierarchy in MySQL snapshot (`cloud_layer_graph_snapshot`) as SSOT for the tree.
2. On each job terminal state, Cloud PutObject the collected `step_full.json` bundle to Tencent COS using key `workspace_{wid}/task_{tid}/comment_{cid}/step_full.json` (admin-overridable pathRule).
3. Hydrate `GET container-job-execution-log` from COS first, then the 023 event table.
4. Expose COS parameters on the platform admin page; secrets only in gitignored local fragment; HTTP client must not use env proxy.
5. Use `cos-go-sdk-v5` with SSE-COS AES256. `backend=local` stores JSON in the pointer table for tests and missing credentials.

## Alternatives Considered

### Alternative 1: MySQL MEDIUMTEXT as authority

- **Pros:** No new infra.
- **Cons:** Large LLM traces bloat the hot DB; cold-hot rules prefer objects for blobs.
- **Why rejected:** User required COS.

### Alternative 2: Container presigned PUT

- **Pros:** Cloud does not proxy bytes.
- **Cons:** Container needs extra CORS/TTL; job JSON already in-process at close.
- **Why rejected:** Server-side PutObject after inbound POST is simpler and keeps secrets off the container.

### Alternative 3: Reuse vendor-docs pathRule as-is

- **Pros:** One COS client.
- **Cons:** Mixes KYC PII keys with task execution logs.
- **Why rejected:** Separate prefix/pathRule; bucket may be shared.

## Consequences

### Positive

- Refresh after container release still shows full steps.
- Admin can change bucket/prefix without redeploying business code (hot-load fragment).

### Negative / Trade-offs

- COS misconfig falls back to local DB blob or 023 summaries.
- Comment-level single key requires merge-on-write for multiple jobs.

### Mitigations

- Idempotent UPSERT by job_id inside the bundle.
- Logs record object_key / bytes only, never secrets.
- pathRule placeholders whitelist; reject `..`.

## References

- `docs/superpowers/specs/2026-08-23-ztree-step-full-cos-archive-design.md`
- [ADR-0006](0006-tencent-cos-vendor-documents.md)
