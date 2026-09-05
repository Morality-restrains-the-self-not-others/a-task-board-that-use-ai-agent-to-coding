# DDD：Git OAuth 资源使用标记

- **日期**: 2026-08-29
- **NFR**: `2026-08-29-git-oauth-resource-grant-marker-nfr-clarification.md`

## Bounded Contexts

| BC | 服务 | 职责 |
|----|------|------|
| GitOauth | taskGitOauth | L1 凭据；state；grant_ticket；回调编排 MarkGrant |
| Project | taskProjectService | 项目 L2 聚合；换票前确认 |
| Task/Comment | taskTaskService | 评论 L2 VO；自动运行作者；排队 UserID |
| Credential/Cloud | taskCredentialService / taskCloudService | 出站换票前问 owner 是否有 L2 |

## Aggregates

### OauthCredentialBinding（已有，不改 PK）

不变。`provider × task2app_user_id × remote_user_id`。

### ProjectGitOauthGrant（新）

- Root: `(project_id, task2app_user_id, gitsite)`
- Attr: `id` Snowflake, `company_id`, `remote_user_id`, `granted_at`
- Invariant: 同一三元组一行；再授权覆盖 remote_user_id
- 不包含 token

### CommentOauthGrant（值对象，挂在 Comment）

- `oauth_gitsite`, `oauth_remote_user_id`, `oauth_granted_at` per repo_identities 项（同 gitsite 共享）
- 作者 `created_by_id` 是换票主体

### GrantTicket（新，GitOauth）

- `id`, `user_id`, `gitsite`, `remote_user_id`, `expires_at`, `consumed_at`
- 一次性；过期不可用

## Ports

```
GrantMarker  Mark(ctx, Grant) error
GrantReader  Has(ctx, resource, user, gitsite) (Grant, bool, error)
TicketIssuer Issue(ctx, user, gitsite, remoteUser) (ticketID, error)
TicketSink   Consume(ctx, ticketID, uid) (GrantDraft, error)
EventBus     Publish(ctx, GrantedEvent) error
```

应用服务 **不得** 在无 Grant 时调用 AccessTokenPort。

## Domain Events

| 事件 | 载荷 | 发布点 | 消费者 |
|------|------|--------|--------|
| `PROJECT_GIT_OAUTH_GRANTED` | project_id, company_id, user_id, gitsite, remote_user_id, granted_at, idempotency_key | Project MarkGrant | 无自动消费（审计）；DLT 不适用直至有 consumer |
| `COMMENT_GIT_OAUTH_GRANTED` | comment_id, task_id, tenant_id, user_id, gitsite, remote_user_id, granted_at, idempotency_key | Task MarkGrant / 发评消费 ticket | 同上 |

幂等键与边界同粒度：`grant:{kind}:{resource_id}:{user_id}:{gitsite}`。禁止 `tenant_id` 单独作消费键。

## 业务意图 → 事件

| 意图 | 事件 | 例外 |
|------|------|------|
| 用户为项目完成 OAuth 回流打标 | PROJECT_GIT_OAUTH_GRANTED | — |
| 用户为评论/自动运行完成打标 | COMMENT_GIT_OAUTH_GRANTED | — |
| 查询徽章 | — | 只读 |
| 换票 | — | 只读出站；失败不另发事件（沿用现网审计表） |

## 文件落点（实现）

- `taskGitOauth/domain/grant_ticket.go` + tests
- `taskProjectService`：grant store + events.go
- `taskTaskService/src/comment_repo_identities.go` 扩展字段
- 不新建微服务
