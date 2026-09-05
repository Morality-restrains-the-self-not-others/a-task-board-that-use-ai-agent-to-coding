# DDD — 项目 L2 种下自动运行评论

- **Date:** 2026-09-01
- **ADR:** 0055（局部取代 ADR-0049 的「永不种下」）

## Bounded contexts

| Context | Owner | 本增量 |
|---------|-------|--------|
| Git OAuth (L1) | taskGitOauth | 不改；project 回调仍不签发 ticket |
| Project | taskProjectService | 查询 ProjectGitOAuthGrant |
| Task / Comment | taskTaskService | AutoRunComment 种下评论 L2 |
| Experience | taskFE | 创建/Fork 门禁 OR |

## Aggregates

- **ProjectGitOAuthGrant**：`(project_id, task2app_user_id, gitsite)` 唯一。只读查询不发事件。
- **AutoRunComment**：`task_comments.repo_identities_json` 上的 oauth_* 字段。种下是对该聚合的一次写入，不是复制 grant 表行。

## Domain events

| 事件 | 何时 | Key |
|------|------|-----|
| `PROJECT_GIT_OAUTH_GRANTED` | 项目页标记（存量） | 不改 |
| `COMMENT_GIT_OAUTH_GRANTED` | ticket 消费或 project_l2_seed | `grant:comment:{commentID}:{userID}:{gitsite}`；payload `via` |

## 不变量

1. 种下仅当 identities 尚缺 `oauth_gitsite`。
2. 查找 `(project_id ∈ 任务关联项目, operator_user_id, gitsite)`。
3. Ticket 优先于 seed。
4. 非自动运行评论不走本路径。
