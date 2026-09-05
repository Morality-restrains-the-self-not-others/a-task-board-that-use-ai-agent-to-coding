# DDD：容器拉取评论级 Git 提交者身份

- **日期**: 2026-08-20

## 限界上下文

| 上下文 | 职责 |
|--------|------|
| CommentRepoIdentity（既有 v83） | 评论级 `{repo_url, git_identity_id}` |
| GitCommitIdentity | `task_git_identities` 的 name/email |
| ContainerCredential | 用 token.CommentID 解析身份给容器 |

## 聚合

- **Comment**：拥有 `repo_identities_json`（真源）。
- **TaskGitIdentity**：值对象，被评论引用，不由容器改写。
- **TaskRepoIdentity**：legacy 任务级绑定，仅 JSON 空时 advisory。

## 不变量

1. 有评论级绑定时，容器看到的 identity_id 必须等于 JSON。
2. name/email 必须来自该 identity 行，不得用作者其它 identity。
3. 嵌套仓可 inherit 已选身份；不得 inherit「作者全部 identity」。
