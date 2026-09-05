# ADR-0010: comment_id 作为 compute 转发 path kv

- **Status:** accepted
- **Date:** 2026-08-16
- **Author:** cursor
- **Deciders:** goal-mode（用户指令：commentId 进 path 而非 query）

---

## Context

评论级 CSC 以 `(task_id, comment_id)` 为键。浏览器经 `taskCloudService` / `taskContainerGateway` 访问容器 compute 时，曾把 `comment_id` 放在 query（`?comment_id=`）。网关 `parseContainerComputePath` 只认 path kv（`tenant_id` / `workspace_id` / `task_id`），未知 key 拒绝；`comment_id` 不在已知 key 中，无法进入 path。query 与 path 作用域分裂，导致执行日志/层图转发 404 或串台。

`shareLib/gatewayauth.ParseConventionPath` 原先也不认 `comment_id`：若插在 `task_id` 与 action 之间，会把后续段当成位置后缀，Cloud 无法把 `compute/container-*` 代理到网关。

## Decision

We will treat **`comment_id` as a first-class convention path kv**, alongside `tenant_id` / `workspace_id` / `task_id`:

1. 浏览器 → SaaS compute URL 形态：

   `/api/cloud/compute/{action}/tenant_id/{t}/workspace_id/{w}/task_id/{task}/comment_id/{cid}/?layer_id=…`

   kv-last 等价：`…/task_id/{task}/comment_id/{cid}/{action}/`。`task_id` **不得**被 `comment_id` 替换。

2. 解析优先级：**path → query → body**（query/body 仅兼容旧客户端）。

3. `ParseConventionPath` 识别 `comment_id` 并写入 `X-Comment-Id`。

4. `layer_id` / `job_id` / `event_id` 等业务过滤参数仍可留在 query。

5. 容器 → SaaS inbound 的 `TaskApiEndPoint` **必须**含位置段 `/comment/{cid}/`：

   `https://<api>/api/tenant/{tid}/workspace/{wid}/task/{taskId}/comment/{cid}/cloud`

   与 token init `/v1/token/init/…/comment/{cid}` 同一位置约定（**不是** compute 的 kv `comment_id`）。SaaS HTTP inbound 必须带该位置段，否则 404。JSON body 的 `comment_id` 仅作双重校验。禁止无 cid 的旧值 `…/task/{taskId}/cloud`。

## Alternatives Considered

### Alternative 1: 继续只用 query `comment_id`

- **Pros:** 前端改动小。
- **Cons:** 与 kv-last 约定不一致；网关 path 解析看不到评论键；易被日志/缓存丢掉。
- **Why rejected:** 用户要求 commentId 进 path。

### Alternative 2: path 用 `comment_id` 替换 `task_id`

- **Pros:** 路径更短。
- **Cons:** CSC 与令牌作用域仍是任务+评论；网关 lookup 与 Cloud 路由都要 `task_id`。
- **Why rejected:** 会破坏现有 parser 与内部 container-target。

## Consequences

### Positive

- 评论级转发 URL 自描述；两评论两实例不再靠 query 对齐。
- Cloud `HasPrefix(compute/container-*)` 与网关 kv 解析一致。

### Negative / Trade-offs

- 旧书签/脚本若只带 query，需兼容期（仅 compute；`TaskApiEndPoint` 不再兼容无 cid 前缀）。
- 内部 `container-target/?comment_id=` 仍为 query（服务间 API，非浏览器）。

### Mitigations

- 网关/云 `commentIDFrom*` 仍读 query/body。
- 前端 helper `appendCommentIdPath` 幂等，禁止再往 compute URL 追加 `?comment_id=`。

## References

- [ADR-0009](0009-comment-level-repo-identity.md) 评论级仓库身份
- [SaaS ↔ 容器 skill 索引](../skills/saas-container/README.md)
- 容器 inbound skill：`trae-agent/onlineServiceJS/skill.md`
- SaaS inbound skill：[saas-machine-container.md](../skills/saas-container/saas-machine-container.md)
