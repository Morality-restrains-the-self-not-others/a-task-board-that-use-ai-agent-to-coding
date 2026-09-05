# ADR-0028: PR 回复评论、远端一键合并与审计

- **Status:** accepted
- **Date:** 2026-08-22
- **Author:** cursor
- **Deciders:** 工程团队（/goal 自动采用）

---

## Context

zTree「推送并创建PR」已能创建 GitHub PR / GitLab MR 并展示 `layer-ztree-pr-btn`，但：

1. 会话时间线不会留下 PR 链接回复；
2. 站内看不到是否已合并；
3. 「合并到目标分支」是容器本地 git merge，不是合并远端 PR；
4. 无「谁点了一键合并」的审计。

跨服务写路径（评论归属 taskTaskService、token 归属 taskGitOauth）需要明确边界。

## Decision

We will:

1. Treat a git PR/MR as a **human comment** nested under the execution comment via path `POST /api/tenant_id/{tid}/workspaceId/{wid}/tasks/{taskId}/comments/{parent_comment_id}/` plus `git_pr_html_url` on `task_comments`. Top-level comment create stays on `/api/tasks/{taskId}/comments/tenant_id/{tid}/`.
2. Deduplicate by `(task_id, git_pr_html_url)` at insert time.
3. Put **status query and merge** on **taskGitOauth**, using the clicking user's OAuth token; GitHub/GitLab remain merge ACL.
4. Record every one-click merge attempt in `git_oauth_taskcredentialaudit` (`action=merge_request_merge`) including user id and html_url.
5. Publish `TASK_GIT_PULL_REQUEST_RECORDED` and `GIT_MERGE_REQUEST_MERGED`.
6. Refresh merge status only on user-triggered loads/actions (no in-process poll).

## Alternatives Considered

### Alternative 1: 只在 zTree PR 按钮旁加状态/合并

- **Pros:** 无评论 schema 变更
- **Cons:** 不满足「创建回复」；刷新层图才看得到
- **Why rejected:** 用户明确要求回复出现在会话中

### Alternative 2: 网关 complete 后 Kafka 建评

- **Pros:** 关页也不丢
- **Cons:** 网关缺少稳定的「当前执行评论」与会话用户；系统作者语义不清
- **Why rejected:** 作者应为点击推送的人；comment_id 前端更准

### Alternative 3: 合并走容器 `container-layer-git-merge`

- **Pros:** 复用现成按钮
- **Cons:** 那是本地分支 merge，不更新 GitLab/GitHub MR 状态
- **Why rejected:** 用户要对「PR 链接」做合并

## Consequences

### Positive

- 评论时间线成为 PR 审查入口
- Token 与审计留在 gitOauth 限界上下文
- 幂等避免重复回复

### Negative / Trade-offs

- 推送成功后立刻关页可能漏评（可用再次推送/手工补评；幂等）
- 状态依赖用户 OAuth；未绑定则显示「无法查询」

### Mitigations

- 创建失败只记 warn，不回滚已成功的 git push
- 合并按钮防重放（同步锁 + Idempotency-Key）+ 服务端按 html_url 去重审计

## References

- 设计: `docs/superpowers/specs/2026-08-22-pr-reply-merge-status-design.md`
- 厂商门户接口说明: https://provider.daydaymoney.com/saas-machine-container （SSOT `docs/skills/saas-container/saas-machine-container.md` §5.2）
- 相关: 意图 `ztree_push_and_create_pr`；ADR-0020 按钮防重放
