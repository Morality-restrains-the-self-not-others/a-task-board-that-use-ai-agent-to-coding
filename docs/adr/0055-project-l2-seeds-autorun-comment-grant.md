# ADR-0055: 项目 Git OAuth L2 可种下同用户自动运行评论 L2

- **Status:** accepted
- **Date:** 2026-09-01
- **Author:** cursor
- **Deciders:** 头脑风暴审批（work-panel-create-task-oauth-after-project-grant）
- **Supersedes (partial):** ADR-0049 中「Project L2 is never copied onto auto-run comments」与「Auto-run cannot ride on project-page OAuth」

---

## Context

ADR-0049 把 L1 凭据与资源 L2 使用标记分开：项目详情 OAuth 写 `project_git_oauth_grant`；创建自动运行只认 pending `grant_ticket`，回调在 `grant_kind=project` 时不签发 ticket。结果是：用户在项目详情已经「已授权」，工作面板创建该项目的自动运行任务仍被门禁拦住。

用户意图（2026-09-01）：同一操作者对**同一项目、同一 Git 站**授过权后，创建该项目下的自动运行任务不应再点一次 OAuth。同事另评、跨项目、仅 L1 无项目标记，仍须拦截。

## Decision

We will allow **seeding** (not sharing a mutable grant row) of the synthetic 【自动运行】 comment L2 from `project_git_oauth_grant` when all of the following match:

- 操作者 = 将写入 `created_by_id` 的人
- `project_id` ∈ 本次创建/Fork 所选项目
- `gitsite` 与仓 URL 主机一致
- 项目行已有 L2（含 `remote_user_id`）

Seeding writes `oauth_gitsite` / `oauth_remote_user_id` / `oauth_granted_at` on **that comment only** and publishes `COMMENT_GIT_OAUTH_GRANTED` with `via=project_l2_seed`.

Priority: consume `grant_ticket` if present; else seed from project L2; else **seed from the source task’s same-user comment L2 when `fork_from_id` is set** (`via=fork_source_comment_l2_seed`); else leave unmarked (FE still blocks submit).

Fork auto-run comments historically posted only `{repo_url, git_identity_id}`. A session `grant_ticket` may already have been consumed on the source comment, so Fork UI can show「已绑定」while the new comment has no `oauth_gitsite`. Copying the **same operator’s** source-comment L2 (by gitsite, including site-level empty `repo_url` rows) is seeding, not sharing a mutable grant row. Other users’ source comments are never copied. Container-snapshot GET repairs unmarked Fork comments on read (idempotent UPDATE).

Create-task / Fork UI treats a repo as bound if session ticket **or** `validate-git-repos` with that `project_id` returns `token_available`.

ADR-0049 的两层模型、换票只认评论作者 + 该评论 L2、同事新评论须自己授权，仍然有效。

## Alternatives Considered

### Alternative 1: Keep ADR-0049, only change copy

- **Pros:** 零后端风险
- **Cons:** 用户仍要点一次；与「项目页已经授过」冲突
- **Why rejected:** 本次明确要求解决二次授权

### Alternative 2: Dual-write grant_ticket on project callback

- **Pros:** 改动面小
- **Cons:** 关页/换设备后 session 丢失，问题复现
- **Why rejected:** 不持久

### Alternative 3: L1 connected implies auto-run OK

- **Pros:** 最少点击
- **Cons:** 复现 ADR-0049 换仓徽章与跨项目误用
- **Why rejected:** 必须仍有项目级标记

## Consequences

### Positive

- 项目详情「已授权」与创建该项目自动运行任务语义对齐
- 评论换票仍有本条 L2，排队/克隆不依赖项目表

### Negative / Trade-offs

- 项目页授权被理解为「允许我在该项目上启用自动运行」
- 多项目任务须每个所选项目对该 gitsite 都有 L2

### Mitigations

- 内部 GET 只读；无项目 L2 不静默用别人的票
- 前端仍对未标记仓展示 pending 绑定链接

## References

- 设计：`docs/superpowers/specs/2026-09-01-work-panel-create-task-oauth-after-project-grant-design.md`
- [ADR-0049](0049-git-oauth-resource-grant-marker.md)
- 意图：`docs/intents/frontend/create_task_oauth_bind.intent.md`
