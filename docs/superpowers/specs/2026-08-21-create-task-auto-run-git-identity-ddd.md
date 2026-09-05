# DDD：创建任务自动运行 Git 提交身份

- **日期**: 2026-08-21

## 限界上下文

`taskTaskService` 任务/评论；`taskFE` / `taskChromePlugin` 为 UI 适配器。无新上下文。

## 值对象

`RepoIdentitySelection{RepoURL, GitIdentityID, GithubUserID}` 已存在。创建自动运行只强制 `RepoURL` + `GitIdentityID`。

## 领域服务

`ValidateGitIdentitiesForCreateAutoRun(selections, requiredRepoURLs)`：auto_run 且 required 非空时每仓须有 git_identity_id。

`ResolveAutoRunRepoIdentities(body, requiredURLs, autoRun)`：auto_run=false 返回空且不报错。

## 聚合

Task 创建成功后生成 Comment（【自动运行】）持有 `repo_identities_json`。不新增表。

## 事件

无新领域事件。TASK_CREATED 不变；身份随评论供 container-snapshot 只读。

## 端口

不新增 Repository。复用 db Exec 写 `task_comments.repo_identities_json`。
