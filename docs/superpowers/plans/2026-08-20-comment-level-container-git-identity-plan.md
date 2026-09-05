# 实现计划：容器拉取评论级 Git 提交者身份

- **日期**: 2026-08-20
- **前置**: NFR + DDD

## Task 1: 端口签名

- [ ] `FetchRepoIdentities(taskID, commentID string)` 所有实现/stub
- **测**: 编译通过；stub 忽略 unused

## Task 2: SQLite 评论 JSON 优先

- [ ] comment_id 非空且 JSON 有项 → lookupGitIdentityDetails
- [ ] 否则 `task_repo_identities`
- **测**: `TestFetchRepoIdentitiesPrefersCommentJSON`

## Task 3: 禁止作者全量覆盖

- [ ] `SelectIdentitiesForCommentAuthor`：已有 per-repo GitIdentityID 则保留
- **测**: 评论选定 gid-A，作者另有 gid-B → 返回 A

## Task 4: 调用点传入 CommentID

- [ ] FetchTaskDetail / BuildRepoCloneCredentials / LayerOauth
- [ ] path comment_id 与 token 不一致 → 403
- **测**: FetchTaskDetail 把 commentID 传给 repo

## Task 5: snapshot 补 name/email + HTTP 映射

- [ ] `repoIdentityMapsFromJSON` 查 `task_git_identities`
- [ ] HTTP FetchRepoIdentities 用 snapshot 而非恒 nil
- **测**: snapshot 含 user_name；HTTP 测映射

## Task 6: 意图文档

- [ ] 更新 `034_comment_level_repo_identity.intent.md` 消费缺口已修
