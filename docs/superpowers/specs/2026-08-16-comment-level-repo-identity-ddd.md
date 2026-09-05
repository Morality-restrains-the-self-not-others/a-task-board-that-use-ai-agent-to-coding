# DDD：评论级仓库身份

> 设计 / NFR / 价值流：2026-08-16-comment-level-repo-identity-*

## 限界上下文

| 上下文 | 服务 | 职责 |
|--------|------|------|
| Task Collaboration | taskTaskService | Comment 聚合、TaskProject 展示绑定 |
| Identity (Git) | taskTaskService `task_git_identities` + GitOAuth | 用户可用署名 / 已连接账号 |
| Cloud Compute | taskCloudService | 用评论身份克隆，不把任务级 binding 当运行真源 |

## 聚合

**Comment（根）**

- 实体：Comment
- 值对象：`RepoIdentitySelection { RepoURL, GitIdentityID, GithubUserID? }`
- 不变量：有 image mention ⇒ 每个任务关联 git_repo 恰好一条 selection 且 GitIdentityID 非空；GitHub URL ⇒ GithubUserID 非空
- 持久化：`repo_identities_json`

**Task**

- 仍拥有 TaskProject（关联项目/分支）
- `task_repo_identities` 不再是运行写入边界（遗留读）

## 领域服务

`ValidateRepoIdentitiesForRun(taskRepos, selections, availableGitIDs, connectedGithubIDs)` → error

纯函数，无 IO。放 `taskTaskService` domain 或 src 纯函数 + 单测。

## 端口

- CommentRepository.Save 含 JSON
- ContainerSnapshotQuery(taskID, commentID) → identities
- 事件总线：既有 publishDomainEvent

## 领域事件

`TASK_COMMENT_IMAGE_MENTIONED` 增补 `repo_identities`（可选数组）。无新事件类型。

## 架构变更影响

见 v83 四类伴生文件。Comment 成为运行身份一致性边界。
