# 容器 task-detail 下发最新子 Git 仓库并并发克隆

日期：2026-07-15  
状态：已批准（goal-mode 自动采用）  
作者：claude

## 问题

项目详情已能发现父仓子 Git 仓库（`nested-git-repos`），但容器引导克隆仍只拿到任务关联的父仓 URL。元仓（如 `ram-work`）下数十个独立子仓不会被克隆。容器侧 `cloneReposIntoSharedLayer` 已用 `Promise.all` 并行，但列表未包含子仓。

## 目标

1. 容器 `POST …/task-detail/` 的 `project_repos[].git_repos` / `git_repo_entries` 含**最新**发现的子仓（`clone_alias`=子路径名）。
2. 同路径 `repo-clone-credentials` 为子仓 URL 返回可用 OAuth 凭证（继承父仓/任务已绑身份）。
3. 容器对列表内仓库**并发**克隆（保留并行；对大批量加并发上限，避免打爆 Git 主机）。
4. 发现失败不阻断父仓克隆。

## 非目标

- 不把子仓持久化进 `project_repos` 表
- 不改 Django 公网路由
- 不改 go_run_container / mock_run_container 启动链路

## 方案（采用）

### A. taskProjectService 内部发现 API

```
GET /api/internal/nested-git-repos/?repo_url=&user_id=
```

复用 `listNestedGitRepos`；供服务间调用（credential / 可选 snapshot）。

### B. taskCredentialService enrich

在 `FetchTaskDetail` 与 `BuildRepoCloneCredentials` 共用：

1. `FetchTaskRepos` + `FetchRepoIdentities`
2. 取首个 `UserID>0` 的身份
3. 对每个项目的每个父仓 URL 调内部 nested API（去重）
4. 将 `nested_repos` 中带有效 `url` 的项 merge 进该项目的 `git_repos` / `git_repo_entries`（`clone_alias=path`；已存在 URL 跳过）
5. 凭证：对无独立 identity 的子仓 URL，**继承**同任务下已有身份（复制 `UserID`/`GitIdentityID`，仅换 `RepoURL`）

响应可另加只读字段 `nested_repos_meta`（可选，MVP 可不加，以免破契约；合并进既有字段即可）。

### C. onlineServiceJS 并发加固

`cloneReposIntoSharedLayer` 已 `Promise.all`。新增：

- 环境变量 `BOOTSTRAP_CLONE_CONCURRENCY`（默认 **8**）信号量限制并行数
- 日志明确「并行克隆 N 仓，并发上限 C」

### 架构

无新微服务；Rel_Flow：Credential → ProjectService internal nested；Credential → 容器契约字段扩展语义。  
架构变更轻量：不强制新 Plateau（与 v28 同迭代延伸）；更新 intents + machine_container.md §4.4。

## 测试

| 层 | 内容 |
|----|------|
| Go Project | internal nested handler |
| Go Credential | merge nested + identity inherit 单测 |
| JS | concurrency pool；collectRepoCloneJobs 含 alias |
| 文档 | intent + machine_container |

## 变更记录

| 日期 | 内容 |
|------|------|
| 2026-07-15 | goal-mode 初版并自动采用 |
