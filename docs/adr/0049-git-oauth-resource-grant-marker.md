# ADR-0049: Git OAuth L1 凭据与项目/评论 L2 使用标记分离

- **Status:** accepted
- **Date:** 2026-08-29
- **Superseded in part by:** [ADR-0055](0055-project-l2-seeds-autorun-comment-grant.md)（2026-09-01：同用户同项目同 gitsite 的项目 L2 **可以种下**【自动运行】评论 L2；两层模型其余条款仍有效）
- **Author:** cursor
- **Deciders:** 头脑风暴审批（git-oauth-resource-grant-marker）

---

## Context

平台对每个 `(provider, 平台用户, remote_user_id)` 只持有一份 OAuth refresh。项目详情用 `user-app-connection` / `token_available` 把「有票」显示成资源「已授权」，换仓后徽章滞留。ADR-0009 规定 OAuth **连接**保持用户级、评论只选 Git 身份。自动运行创建门禁也只看 L1。排队调度 `UserID=t.OwnerID` 与【自动运行】评论作者可能不是同一人。

需要：连接仍一份；**使用**必须在本站对项目或评论打标；项目本身不会自动运行。

## Decision

We will keep L1 credentials in `taskGitOauth` (`git_oauth_appusercredential`) unchanged in granularity.

We will add **L2 usage grants** owned by resource services:

- Project page (cloud-dev / preview): `project_git_oauth_grant` in `taskProjectService`.
- Comment run / push / auto-run: fields on `task_comments.repo_identities_json` in `taskTaskService`.

OAuth callback writes L2 for the `grant_kind`/`grant_id` in state (or a one-shot ticket if the comment does not exist yet). Callers must check L2 before `access-for-user`.

Git actor is the **person who enabled auto-run or posted the comment**, not `owner_id` / `operator_id`. Queue start must use the auto-run comment `created_by_id`. Project L2 is never copied onto auto-run comments.

ADR-0009 §4 remains for **connection** (L1). Usage gating is this ADR.

## Alternatives Considered

### Alternative 1: Per-repo refresh tokens

- **Pros:** Matches user intuition of「绑这个仓」
- **Cons:** Violates one-credential-per-remote-user; GitHub OAuth is user-to-server
- **Why rejected:** Product keeps a single L1 row

### Alternative 2: L1 connected implies all resources authorized

- **Pros:** Current code; fewer clicks
- **Cons:** Wrong badge after repo switch; auto-run impersonates any project
- **Why rejected:** Explicit user request for resource markers

### Alternative 3: Auto-run uses task Owner token

- **Pros:** Assignee can run after handoff
- **Cons:** Create gate checks creator; clone uses comment author — mismatch
- **Why rejected:** User locked Git to enabler / comment author; re-confirmed 2026-08-29 questionnaire `enabler_must_oauth`

## Consequences

### Positive

- 「已授权」means this resource is marked and writable
- Auto-run cannot ride on project-page OAuth
- Owner reassignment does not silently switch Git identity

### Negative / Trade-offs

- Users re-OAuth per project page and per comment (IdP often instant)
- No L2 backfill — breaking UX on ship
- Extra internal MarkGrant APIs and Kafka events

### Mitigations

- Grant ticket for create-task auto-run before comment id exists
- Probe push still distinguishes marked-but-no-write

## References

- 设计：`docs/superpowers/specs/2026-08-29-git-oauth-resource-grant-marker-design.md`
- ADR-0009 评论级仓库身份
- 架构 v118 target
