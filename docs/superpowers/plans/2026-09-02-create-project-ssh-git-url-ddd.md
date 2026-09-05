# DDD — GitRepoURL 值对象扩展 ssh://

- **Date:** 2026-09-02
- **Bounded context:** Project (taskProjectService) + taskFE create/edit project

## Value object: GitRepoURL

合法远程：

- empty (optional)
- `http(s)://host/path`
- `git@host:path` (SCP)
- `ssh://[user@]host[:port]/path`

操作：

- `IsValid` — 前端 `isValidGitRepoUrl`
- `NormalizeForGitHTTPAPI` — `normalizeGitRepoURLForBranchLookup`（OAuth/分支）
- `MatchKey` — `gitCloneRefMatchKey`（host+path，忽略 scheme/.git）

无新实体、无新聚合、无新领域事件。意图「填写 ssh URL」不是独立业务状态变更。

## Ports

沿用现有 validate-git-repos / project write；不新增端口。
