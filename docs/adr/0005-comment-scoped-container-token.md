# ADR-0005: 容器令牌按评论隔离

- **Status:** accepted
- **Date:** 2026-08-14
- **Author:** cursor
- **Deciders:** goal-mode（用户指令：把令牌改为评论级的）

---

## Context

运行时已是一评论一容器，但 `credential_container_tokens` 聚合标识仍是 `TaskScope(tenant, workspace, task)`。同一 Task 下先启动的容器 `exchange-refresh` 成功后清空预埋 access；后到评论容器拿到同一份 `ACCESS_TOKEN`、本地无 `container_refresh_token.json`，得到 403 `TOKEN_EXCHANGE_ALREADY_DONE` 后 fail-closed。幂等换票只能自愈「同一预埋 access 重建容器」，无法隔离两个评论。

## Decision

We will treat container token identity as `(tenant_id, workspace_id, task_id, comment_id)`:

1. `TaskScope` 增加 `CommentID`；`IssueToken` 必须带非空 `comment_id`。
2. 表 `credential_container_tokens` 增加 `comment_id`，去重后 `UNIQUE(company_id, workspace_id, task_id, comment_id)`。
3. Init 路径：`POST /v1/token/init/tenant/{t}/workspace/{w}/task/{tk}/comment/{c}`；旧 9 段路径须 query/body 补 `comment_id`，否则 400。
4. `GET /v1/token/by-scope` 增加 `comment_id`；缺省时仅当该 task 下唯一一行才回退，多行 400/404。
5. `exchange-refresh` / `refresh-access` body 可选 `comment_id`；双方均非空且不等 → `SCOPE_MISMATCH`。URL 无 comment 的旧容器仍可按 access 命中行。
6. 同一评论再次 `IssueToken`：复用该行；已有 refresh 时只换 access、不清 refresh（避免插空 refresh 新行）。

DDL 只放 `dataMigrate/taskCredentialService/`，业务进程不迁移。

## Alternatives Considered

### Alternative 1: 仅幂等 exchange-refresh

- **Pros:** 改动小，已热修过重建容器。
- **Cons:** 两评论仍共享一行，后到容器无法独立换票。
- **Why rejected:** 用户明确要求评论级令牌。

### Alternative 2: 新聚合名 CommentScope 替换 TaskScope

- **Pros:** 命名与一致性边界更贴切。
- **Cons:** 全仓重命名，容器 URL 路径仍无 comment 段，兼容成本高。
- **Why rejected:** expand/contract：保留 `TaskScope` 字段扩展即可。

## Consequences

### Positive

- 同一 Task 两评论各有独立 access/refresh，互不抢票。
- 与已有评论级 CSC / 容器名契约对齐。

### Negative / Trade-offs

- 存量 `comment_id=''` 行视为任务级遗留；新签发强制 comment。
- 旧镜像不传 `comment_id` 时依赖 access 命中 + Cloud CSC 回退。

### Mitigations

- ValidateToken：仅当双方 comment 均非空且不等才 SCOPE_MISMATCH。
- onlineServiceJS 换票/登记带 `COMMENT_ID`。

## References

- `docs/architecture/06_container_token_ddd_design.md`
- `docs/superpowers/specs/2026-08-14-comment-scoped-container-token-design.md`
- ADR-0001 / ADR-0002（DDL 与业务进程解耦）
